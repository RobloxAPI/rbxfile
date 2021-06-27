package rbxl

import (
	"errors"
	"fmt"
	"sort"

	"github.com/robloxapi/rbxfile"
	"golang.org/x/crypto/blake2b"
)

// RobloxCodec implements Decoder and Encoder to emulate Roblox's internal
// codec as closely as possible.
type RobloxCodec struct {
	Mode Mode
}

// Reference value indicating a nil instance.
const nilInstance = -1

func (c RobloxCodec) Decode(model *formatModel) (root *rbxfile.Root, err error) {
	if model == nil {
		return nil, fmt.Errorf("FormatModel is nil")
	}
	model.Warnings = model.Warnings[:0]

	root = new(rbxfile.Root)

	groupLookup := make(map[int32]*chunkInstance, model.ClassCount)
	instLookup := make(map[int32]*rbxfile.Instance, model.InstanceCount+1)
	instLookup[nilInstance] = nil

	var sharedStrings []SharedString
	var chunkType string
	var chunkNum int

	addWarn := func(format string, v ...interface{}) {
		q := make([]interface{}, 0, len(v)+2)
		q = append(q, chunkType)
		q = append(q, chunkNum)
		q = append(q, v...)
		model.Warnings = append(model.Warnings, fmt.Errorf("%s chunk (#%d): "+format, q...))
	}

loop:
	for ic, chunk := range model.Chunks {
		chunkNum = ic
		switch chunk := chunk.(type) {
		case *chunkInstance:
			chunkType = "instance"
			if chunk.ClassID < 0 || uint32(chunk.ClassID) >= model.ClassCount {
				err = fmt.Errorf("class index out of bounds: %d", model.ClassCount)
				goto chunkErr
			}
			// No error if ClassCount > actual count.

			if chunk.IsService && len(chunk.InstanceIDs) != len(chunk.GetService) {
				err = fmt.Errorf("malformed instance chunk (class ID %d): GetService array length does not equal InstanceIDs array length", chunk.ClassID)
				goto chunkErr
			}

			for i, ref := range chunk.InstanceIDs {
				if ref < 0 || uint32(ref) >= model.InstanceCount {
					err = fmt.Errorf("invalid id %d", ref)
					goto chunkErr
				}
				// No error if InstanceCount > actual count.

				inst := rbxfile.NewInstance(chunk.ClassName)
				if _, ok := instLookup[ref]; ok {
					err = fmt.Errorf("duplicate id: %d", ref)
					goto chunkErr
				}

				if chunk.IsService && chunk.GetService[i] == 1 {
					inst.IsService = true
				}

				instLookup[ref] = inst
			}

			if _, ok := groupLookup[chunk.ClassID]; ok {
				err = fmt.Errorf("duplicate class index: %d", chunk.ClassID)
				goto chunkErr
			}
			groupLookup[chunk.ClassID] = chunk

		case *chunkProperty:
			chunkType = "property"
			if chunk.ClassID < 0 || uint32(chunk.ClassID) >= model.ClassCount {
				err = fmt.Errorf("class index out of bounds: %d", model.ClassCount)
				goto chunkErr
			}
			// No error if TypeCount > actual count.

			instChunk, ok := groupLookup[chunk.ClassID]
			if !ok {
				addWarn("class `%d` of property group is invalid or unknown", chunk.ClassID)
				continue
			}

			if len(chunk.Properties) != len(instChunk.InstanceIDs) {
				err = fmt.Errorf("length of properties array (%d) does not equal length of class array (%d)", len(chunk.Properties), len(instChunk.InstanceIDs))
				goto chunkErr
			}

			for i, bvalue := range chunk.Properties {
				inst := instLookup[instChunk.InstanceIDs[i]]
				var value rbxfile.Value
				switch bvalue := bvalue.(type) {
				case *ValueReference:
					value = rbxfile.ValueReference{Instance: instLookup[int32(*bvalue)]}
				case *ValueSharedString:
					i := int(*bvalue)
					if i < 0 || i >= len(sharedStrings) {
						// TODO: How are invalid indexes handled?
						value = rbxfile.ValueSharedString("")
						break
					}
					value = rbxfile.ValueSharedString(sharedStrings[i].Value)
				default:
					value = DecodeValue(bvalue)
				}
				inst.Properties[chunk.PropertyName] = value
			}

		case *chunkParent:
			chunkType = "parent"
			if chunk.Version != 0 {
				err = fmt.Errorf("unrecognized parent link format %d", chunk.Version)
				goto chunkErr
			}

			if len(chunk.Parents) != len(chunk.Children) {
				err = fmt.Errorf("length of Parents array does not equal length of Children array")
				goto chunkErr
			}

			for i, ref := range chunk.Children {
				if ref < 0 || uint32(ref) >= model.InstanceCount {
					err = fmt.Errorf("invalid id %d", ref)
					goto chunkErr
				}

				child := instLookup[ref]
				if child == nil {
					addWarn("referent #%d `%d` does not exist", i, ref)
					continue
				}

				if chunk.Parents[i] == nilInstance {
					root.Instances = append(root.Instances, child)
					continue
				}

				parent, ok := instLookup[chunk.Parents[i]]
				// RESEARCH: overriding with a nil referent vs non-existent referent.
				if !ok {
					continue
				}

				parent.Children = append(parent.Children, child)
			}

		case *chunkMeta:
			chunkType = "meta"
			if root.Metadata == nil {
				root.Metadata = make(map[string]string, len(chunk.Values))
			}
			for _, pair := range chunk.Values {
				root.Metadata[pair[0]] = pair[1]
			}

		case *chunkSharedStrings:
			chunkType = "sharedstring"
			// TODO: How are multiple chunks handled (overwrite or append)?
			sharedStrings = chunk.Values

		case *chunkEnd:
			chunkType = "end"
			break loop
		}
	}

	return

chunkErr:
	err = fmt.Errorf("%s chunk (#%d): %s", chunkType, chunkNum, err)
	return nil, err
}

// DecodeValue converts a Value to a rbxfile.Value. Returns nil if the value
// could not be decoded.
//
// ValueString is always converted to a rbxfile.ValueString.
//
// ValueReference and ValueSharedString, which require external information in
// order to decode, return nil.
func DecodeValue(value Value) rbxfile.Value {
	switch value := value.(type) {
	case *ValueString:
		v := make([]byte, len(*value))
		copy(v, *value)
		// The binary format does not differentiate between the various string
		// types.
		return rbxfile.ValueString(v)

	case *ValueBool:
		return rbxfile.ValueBool(*value)

	case *ValueInt:
		return rbxfile.ValueInt(*value)

	case *ValueFloat:
		return rbxfile.ValueFloat(*value)

	case *ValueDouble:
		return rbxfile.ValueDouble(*value)

	case *ValueUDim:
		return rbxfile.ValueUDim{
			Scale:  float32(value.Scale),
			Offset: int32(value.Offset),
		}

	case *ValueUDim2:
		return rbxfile.ValueUDim2{
			X: rbxfile.ValueUDim{
				Scale:  float32(value.ScaleX),
				Offset: int32(value.OffsetX),
			},
			Y: rbxfile.ValueUDim{
				Scale:  float32(value.ScaleY),
				Offset: int32(value.OffsetY),
			},
		}

	case *ValueRay:
		return rbxfile.ValueRay{
			Origin: rbxfile.ValueVector3{
				X: value.OriginX,
				Y: value.OriginY,
				Z: value.OriginZ,
			},
			Direction: rbxfile.ValueVector3{
				X: value.DirectionX,
				Y: value.DirectionY,
				Z: value.DirectionZ,
			},
		}

	case *ValueFaces:
		return rbxfile.ValueFaces(*value)

	case *ValueAxes:
		return rbxfile.ValueAxes(*value)

	case *ValueBrickColor:
		return rbxfile.ValueBrickColor(*value)

	case *ValueColor3:
		return rbxfile.ValueColor3{
			R: float32(value.R),
			G: float32(value.G),
			B: float32(value.B),
		}

	case *ValueVector2:
		return rbxfile.ValueVector2{
			X: float32(value.X),
			Y: float32(value.Y),
		}

	case *ValueVector3:
		return rbxfile.ValueVector3{
			X: float32(value.X),
			Y: float32(value.Y),
			Z: float32(value.Z),
		}

	case *ValueVector2int16:
		return rbxfile.ValueVector2int16(*value)

	case *ValueCFrame:
		cf := rbxfile.ValueCFrame{
			Position: rbxfile.ValueVector3{
				X: float32(value.Position.X),
				Y: float32(value.Position.Y),
				Z: float32(value.Position.Z),
			},
			Rotation: value.Rotation,
		}

		if value.Special != 0 {
			cf.Rotation = cframeSpecialMatrix[value.Special]
		}

		return cf

	case *ValueCFrameQuat:
		v := value.ToCFrame()
		cf := rbxfile.ValueCFrame{
			Position: rbxfile.ValueVector3{
				X: float32(v.Position.X),
				Y: float32(v.Position.Y),
				Z: float32(v.Position.Z),
			},
			Rotation: v.Rotation,
		}

		if v.Special != 0 {
			cf.Rotation = cframeSpecialMatrix[v.Special]
		}

		return cf

	case *ValueToken:
		return rbxfile.ValueToken(*value)

	case *ValueReference:
		// Must be resolved elsewhere.
		return nil

	case *ValueVector3int16:
		return rbxfile.ValueVector3int16(*value)

	case *ValueNumberSequence:
		v := make(rbxfile.ValueNumberSequence, len(*value))
		for i, nsk := range *value {
			v[i] = rbxfile.ValueNumberSequenceKeypoint{
				Time:     nsk.Time,
				Value:    nsk.Value,
				Envelope: nsk.Envelope,
			}
		}
		return v

	case *ValueColorSequence:
		v := make(rbxfile.ValueColorSequence, len(*value))
		for i, nsk := range *value {
			v[i] = rbxfile.ValueColorSequenceKeypoint{
				Time: nsk.Time,
				Value: rbxfile.ValueColor3{
					R: float32(nsk.Value.R),
					G: float32(nsk.Value.G),
					B: float32(nsk.Value.B),
				},
				Envelope: nsk.Envelope,
			}
		}
		return v

	case *ValueNumberRange:
		return rbxfile.ValueNumberRange(*value)

	case *ValueRect:
		return rbxfile.ValueRect{
			Min: rbxfile.ValueVector2{
				X: float32(value.Min.X),
				Y: float32(value.Min.Y),
			},
			Max: rbxfile.ValueVector2{
				X: float32(value.Max.X),
				Y: float32(value.Max.Y),
			},
		}

	case *ValuePhysicalProperties:
		return rbxfile.ValuePhysicalProperties{
			CustomPhysics:    value.CustomPhysics != 0,
			Density:          value.Density,
			Friction:         value.Friction,
			Elasticity:       value.Elasticity,
			FrictionWeight:   value.FrictionWeight,
			ElasticityWeight: value.ElasticityWeight,
		}

	case *ValueColor3uint8:
		return rbxfile.ValueColor3uint8{
			R: value.R,
			G: value.G,
			B: value.B,
		}

	case *ValueInt64:
		return rbxfile.ValueInt64(*value)

	case *ValueSharedString:
		// Must be resolved elsewhere.
		return nil

	default:
		return nil
	}
}

type sharedEntry struct {
	index int
	value SharedString
}

type sharedMap map[[16]byte]sharedEntry

func (c RobloxCodec) Encode(root *rbxfile.Root) (model *formatModel, err error) {
	if root == nil {
		return nil, errors.New("Root is nil")
	}

	model = new(formatModel)

	// A list of instances in the tree. The index serves as the instance's
	// reference number.
	instList := make([]*rbxfile.Instance, 0)

	// A map used to ensure that an instance is counted only once. Also used
	// to link ValueReferences.
	refs := map[*rbxfile.Instance]int{
		nil: nilInstance,
	}

	// Set of shared strings mapped to indexes.
	sharedStrings := sharedMap{}

	// Recursively finds and adds instances.
	var addInstance func(inst *rbxfile.Instance)
	addInstance = func(inst *rbxfile.Instance) {
		if _, ok := refs[inst]; ok {
			// Ignore the instance if it has already been read.
			return
		}

		// Reference number should match position in list.
		refs[inst] = len(instList)
		instList = append(instList, inst)

		for _, child := range inst.Children {
			addInstance(child)
		}
	}

	// For RBXL, each instance in the Root is an instance in the DataModel.
	// For RBXM, each instance in the Root is an instance in the selection.
	for _, inst := range root.Instances {
		addInstance(inst)
	}

	// Group instances of the same ClassName into single chunks.
	instChunkMap := map[string]*chunkInstance{}
	for ref, inst := range instList {

		chunk, ok := instChunkMap[inst.ClassName]
		if !ok {
			chunk = &chunkInstance{
				IsCompressed: true,
				ClassName:    inst.ClassName,
				InstanceIDs:  []int32{},
			}
			instChunkMap[inst.ClassName] = chunk
		}

		chunk.InstanceIDs = append(chunk.InstanceIDs, int32(ref))

		if c.Mode == Place && inst.IsService {
			chunk.IsService = true
			chunk.GetService = append(chunk.GetService, 1)
		} else {
			chunk.GetService = append(chunk.GetService, 0)
		}
	}

	// Sort chunks by ClassName.
	instChunkList := make(sortInstChunks, len(instChunkMap))
	if len(instChunkMap) > 0 {
		classID := 0
		for _, chunk := range instChunkMap {
			instChunkList[classID] = chunk
			classID++
		}

		sort.Sort(instChunkList)
	}

	// Make property chunks.
	propChunkList := []*chunkProperty{}
	for i, instChunk := range instChunkList {
		instChunk.ClassID = int32(i)

		addWarn := func(format string, v ...interface{}) {
			q := make([]interface{}, 0, len(v)+1)
			q = append(q, instChunk.ClassID)
			q = append(q, v...)
			model.Warnings = append(model.Warnings, fmt.Errorf("instance chunk #%d: "+format, q...))
		}

		propChunkMap := map[string]*chunkProperty{}

		// Populate propChunkMap.
		for _, ref := range instChunk.InstanceIDs {
			for name := range instList[ref].Properties {
				if _, ok := propChunkMap[name]; ok {
					// A chunk of the property name already exists.
					continue
				}
				propChunkMap[name] = &chunkProperty{
					IsCompressed: true,
					ClassID:      instChunk.ClassID,
					PropertyName: name,
					Properties:   make([]Value, len(instChunk.InstanceIDs)),
				}
			}
		}

		// Check to see if all existing properties types match.
	checkPropType:
		for name, propChunk := range propChunkMap {
			var instRef int32 = nilInstance
			dataType := TypeInvalid
			for _, ref := range instChunk.InstanceIDs {
				inst := instList[ref]
				prop, ok := inst.Properties[name]
				if !ok {
					continue
				}
				if dataType == TypeInvalid {
					// Set data type to the first valid property.
					dataType = FromValueType(prop.Type())
					if dataType == TypeInvalid {
						addWarn("unknown type %d for property %s.%s in instance #%d, chunk skipped", byte(dataType), instList[instRef].ClassName, name, instRef)
						continue checkPropType
					}
					instRef = ref
					continue
				}
				if t := FromValueType(prop.Type()); t != dataType {
					// If at least one property type does not match with the
					// rest, then stop.
					delete(propChunkMap, name)
					addWarn("mismatched types %s and %s for property %s.%s, chunk skipped", t, dataType, instList[instRef].ClassName, name)
					continue checkPropType
				}
			}
			// Because propChunkMap was populated from InstanceIDs, dataType
			// should always be a valid value by this point.
			propChunk.DataType = dataType
		}

		// Set the values for each property chunk.
		for name, propChunk := range propChunkMap {
			for i, ref := range instChunk.InstanceIDs {
				inst := instList[ref]

				var bvalue Value
				if value, ok := inst.Properties[name]; ok {
					switch value := value.(type) {
					case rbxfile.ValueReference:
						// Convert an instance reference to a reference number.
						ref, ok := refs[value.Instance]
						if !ok {
							// References that map to some instance not under the
							// Root should be nil.
							ref = nilInstance
						}

						v := int32(ref)
						bvalue = (*ValueReference)(&v)
					case rbxfile.ValueSharedString:
						// TODO: verify that strings are compared by hash.
						sum := blake2b.Sum256([]byte(value))
						var hash [16]byte
						copy(hash[:], sum[:])
						entry, ok := sharedStrings[hash]
						if !ok {
							entry.index = len(sharedStrings)
							// No longer used; Roblox encodes with zeros.
							entry.value.Hash = [16]byte{}
							entry.value.Value = []byte(value)
							sharedStrings[hash] = entry
						}
						index := uint32(entry.index)
						bvalue = (*ValueSharedString)(&index)
					default:
						bvalue = EncodeValue(value)
					}
				}

				if bvalue == nil || bvalue.Type() != propChunk.DataType {
					// Use default value for DataType.
					bvalue = NewValue(propChunk.DataType)
				}

				propChunk.Properties[i] = bvalue
			}
		}

		// Sort the chunks by PropertyName.
		propChunks := make(sortPropChunks, len(propChunkMap))
		if len(propChunkMap) > 0 {
			i := 0
			for _, chunk := range propChunkMap {
				propChunks[i] = chunk
				i++
			}

			sort.Sort(propChunks)
		}

		propChunkList = append(propChunkList, propChunks...)
	}

	// Make parent chunk.
	parentChunk := &chunkParent{
		IsCompressed: true,
		Version:      0,
		Children:     make([]int32, len(instList)),
		Parents:      make([]int32, len(instList)),
	}

	if len(instList) > 0 {
		i := 0
		var recInst func(inst, parent *rbxfile.Instance)
		recInst = func(inst, parent *rbxfile.Instance) {
			for _, child := range inst.Children {
				recInst(child, inst)
			}

			instRef := int32(refs[inst])
			parentChunk.Children[i] = instRef
			parentRef, ok := refs[parent]
			if !ok {
				parentRef = nilInstance
			}
			parentChunk.Parents[i] = int32(parentRef)
			i++
		}
		for _, inst := range root.Instances {
			recInst(inst, nil)
		}
	}

	// Make end chunk.
	endChunk := &chunkEnd{
		IsCompressed: false,
		Content:      []byte("</roblox>"),
	}

	// Make FormatModel.
	model.ClassCount = uint32(len(instChunkList))
	model.InstanceCount = uint32(len(instList))

	chunkLength := len(instChunkList) + len(propChunkList) + 1
	if len(root.Metadata) > 0 {
		chunkLength++
	}
	if len(sharedStrings) > 0 {
		chunkLength++
	}
	model.Chunks = make([]chunk, 0, chunkLength)

	if len(root.Metadata) > 0 {
		// TODO: verify that chunk is omitted when zero values are encoded, and
		// is not based on format (RBXM vs RBXL).
		chunk := chunkMeta{
			IsCompressed: true,
			Values:       make([][2]string, 0, len(root.Metadata)),
		}
		for key, value := range root.Metadata {
			chunk.Values = append(chunk.Values, [2]string{key, value})
		}
		sort.Sort(sortMetaData(chunk.Values))
		model.Chunks = append(model.Chunks, &chunk)
	}

	if len(sharedStrings) > 0 {
		chunk := chunkSharedStrings{
			Version: 0,
			Values:  make([]SharedString, len(sharedStrings)),
		}
		for _, entry := range sharedStrings {
			chunk.Values[entry.index] = entry.value
		}
		model.Chunks = append(model.Chunks, &chunk)
	}

	for _, chunk := range instChunkList {
		model.Chunks = append(model.Chunks, chunk)
	}
	for _, chunk := range propChunkList {
		model.Chunks = append(model.Chunks, chunk)
	}
	model.Chunks = append(model.Chunks, parentChunk)
	model.Chunks = append(model.Chunks, endChunk)

	return
}

type sortInstChunks []*chunkInstance

func (c sortInstChunks) Len() int {
	return len(c)
}
func (c sortInstChunks) Less(i, j int) bool {
	return c[i].ClassName < c[j].ClassName
}
func (c sortInstChunks) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type sortPropChunks []*chunkProperty

func (c sortPropChunks) Len() int {
	return len(c)
}
func (c sortPropChunks) Less(i, j int) bool {
	return c[i].PropertyName < c[j].PropertyName
}
func (c sortPropChunks) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type sortMetaData [][2]string

func (c sortMetaData) Len() int {
	return len(c)
}
func (c sortMetaData) Less(i, j int) bool {
	return c[i][0] < c[j][1]
}
func (c sortMetaData) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// EncodeValue converts a rbxfile.Value to a Value. Returns nil if the value
// could not be encoded.
//
// Because the rbxl format has only one string type, the following types are
// converted to ValueString:
//
//     - rbxfile.ValueString
//     - rbxfile.ValueBinaryString
//     - rbxfile.ValueProtectedString
//     - rbxfile.ValueContent
//
// rbxfile.ValueReference and rbxfile.ValueSharedString, which require external
// information in order to encode, return nil.
func EncodeValue(value rbxfile.Value) Value {
	switch value := value.(type) {
	case rbxfile.ValueString:
		v := make([]byte, len(value))
		copy(v, value)
		return (*ValueString)(&v)

	case rbxfile.ValueBinaryString:
		v := make([]byte, len(value))
		copy(v, value)
		return (*ValueString)(&v)

	case rbxfile.ValueProtectedString:
		v := make([]byte, len(value))
		copy(v, value)
		return (*ValueString)(&v)

	case rbxfile.ValueContent:
		v := make([]byte, len(value))
		copy(v, value)
		return (*ValueString)(&v)

	case rbxfile.ValueBool:
		return (*ValueBool)(&value)

	case rbxfile.ValueInt:
		return (*ValueInt)(&value)

	case rbxfile.ValueFloat:
		return (*ValueFloat)(&value)

	case rbxfile.ValueDouble:
		return (*ValueDouble)(&value)

	case rbxfile.ValueUDim:
		return &ValueUDim{
			Scale:  ValueFloat(value.Scale),
			Offset: ValueInt(value.Offset),
		}

	case rbxfile.ValueUDim2:
		return &ValueUDim2{
			ScaleX:  ValueFloat(value.X.Scale),
			ScaleY:  ValueFloat(value.Y.Scale),
			OffsetX: ValueInt(value.X.Offset),
			OffsetY: ValueInt(value.Y.Offset),
		}

	case rbxfile.ValueRay:
		return &ValueRay{
			OriginX:    value.Origin.X,
			OriginY:    value.Origin.Y,
			OriginZ:    value.Origin.Z,
			DirectionX: value.Direction.X,
			DirectionY: value.Direction.Y,
			DirectionZ: value.Direction.Z,
		}

	case rbxfile.ValueFaces:
		return (*ValueFaces)(&value)

	case rbxfile.ValueAxes:
		return (*ValueAxes)(&value)

	case rbxfile.ValueBrickColor:
		return (*ValueBrickColor)(&value)

	case rbxfile.ValueColor3:
		return &ValueColor3{
			R: ValueFloat(value.R),
			G: ValueFloat(value.G),
			B: ValueFloat(value.B),
		}

	case rbxfile.ValueVector2:
		return &ValueVector2{
			X: ValueFloat(value.X),
			Y: ValueFloat(value.Y),
		}

	case rbxfile.ValueVector3:
		return &ValueVector3{
			X: ValueFloat(value.X),
			Y: ValueFloat(value.Y),
			Z: ValueFloat(value.Z),
		}

	case rbxfile.ValueCFrame:
		cf := &ValueCFrame{
			Position: ValueVector3{
				X: ValueFloat(value.Position.X),
				Y: ValueFloat(value.Position.Y),
				Z: ValueFloat(value.Position.Z),
			},
		}

		if s, ok := cframeSpecialNumber[value.Rotation]; ok {
			cf.Special = s
		} else {
			cf.Rotation = value.Rotation
		}

		return cf

	case rbxfile.ValueToken:
		return (*ValueToken)(&value)

	case rbxfile.ValueReference:
		// Must be resolved elsewhere.
		return nil

	case rbxfile.ValueVector3int16:
		return (*ValueVector3int16)(&value)

	case rbxfile.ValueVector2int16:
		return (*ValueVector2int16)(&value)

	case rbxfile.ValueNumberSequence:
		v := make(ValueNumberSequence, len(value))
		for i, nsk := range value {
			v[i] = ValueNumberSequenceKeypoint{
				Time:     nsk.Time,
				Value:    nsk.Value,
				Envelope: nsk.Envelope,
			}
		}
		return &v

	case rbxfile.ValueColorSequence:
		v := make(ValueColorSequence, len(value))
		for i, nsk := range value {
			v[i] = ValueColorSequenceKeypoint{
				Time: nsk.Time,
				Value: ValueColor3{
					R: ValueFloat(nsk.Value.R),
					G: ValueFloat(nsk.Value.G),
					B: ValueFloat(nsk.Value.B),
				},
				Envelope: nsk.Envelope,
			}
		}
		return &v

	case rbxfile.ValueNumberRange:
		return (*ValueNumberRange)(&value)

	case rbxfile.ValueRect:
		return &ValueRect{
			Min: ValueVector2{
				X: ValueFloat(value.Min.X),
				Y: ValueFloat(value.Min.Y),
			},
			Max: ValueVector2{
				X: ValueFloat(value.Max.X),
				Y: ValueFloat(value.Max.Y),
			},
		}

	case rbxfile.ValuePhysicalProperties:
		v := ValuePhysicalProperties{
			Density:          value.Density,
			Friction:         value.Friction,
			Elasticity:       value.Elasticity,
			FrictionWeight:   value.FrictionWeight,
			ElasticityWeight: value.ElasticityWeight,
		}
		if value.CustomPhysics {
			v.CustomPhysics = 1
		} else {
			v.CustomPhysics = 0
		}
		return &v

	case rbxfile.ValueColor3uint8:
		return &ValueColor3uint8{
			R: value.R,
			G: value.G,
			B: value.B,
		}

	case rbxfile.ValueInt64:
		return (*ValueInt64)(&value)

	case rbxfile.ValueSharedString:
		// Must be resolved elsewhere.
		return nil

	default:
		return nil
	}
}
