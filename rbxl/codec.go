package rbxl

import (
	"crypto/md5"
	"errors"
	"fmt"
	"sort"

	"github.com/robloxapi/rbxfile"
)

// Mode indicates how RobloxCodec should interpret data.
type Mode uint8

const (
	ModePlace Mode = iota // Data is decoded and encoded as a Roblox place (RBXL) file.
	ModeModel             // Data is decoded and encoded as a Roblox model (RBXM) file.
)

// RobloxCodec implements Decoder and Encoder to emulate Roblox's internal
// codec as closely as possible.
type RobloxCodec struct {
	Mode Mode
}

func (c RobloxCodec) Decode(model *FormatModel) (root *rbxfile.Root, err error) {
	if model == nil {
		return nil, fmt.Errorf("FormatModel is nil")
	}
	model.Warnings = model.Warnings[:0]

	root = new(rbxfile.Root)

	groupLookup := make(map[int32]*ChunkInstance, model.ClassCount)
	instLookup := make(map[int32]*rbxfile.Instance, model.InstanceCount+1)
	instLookup[-1] = nil

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
		case *ChunkInstance:
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

		case *ChunkProperty:
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
					value = decodeValue(bvalue)
				}
				inst.Properties[chunk.PropertyName] = value
			}

		case *ChunkParent:
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

				if chunk.Parents[i] == -1 {
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

		case *ChunkMeta:
			chunkType = "meta"
			if root.Metadata == nil {
				root.Metadata = make(map[string]string, len(chunk.Values))
			}
			for _, pair := range chunk.Values {
				root.Metadata[pair[0]] = pair[1]
			}

		case *ChunkSharedStrings:
			chunkType = "sharedstring"
			// TODO: How are multiple chunks handled (overwrite or append)?
			sharedStrings = chunk.Values

		case *ChunkEnd:
			chunkType = "end"
			break loop
		}
	}

	return

chunkErr:
	err = fmt.Errorf("%s chunk (#%d): %s", chunkType, chunkNum, err)
	return nil, err
}

// Decode a rbxl.value to a rbxfile.Value based on a given value type.
func decodeValue(bvalue Value) (value rbxfile.Value) {
	switch bvalue := bvalue.(type) {
	case *ValueString:
		v := make([]byte, len(*bvalue))
		copy(v, *bvalue)
		// The binary format does not differentiate between the various string
		// types.
		value = rbxfile.ValueString(v)

	case *ValueBool:
		value = rbxfile.ValueBool(*bvalue)

	case *ValueInt:
		value = rbxfile.ValueInt(*bvalue)

	case *ValueFloat:
		value = rbxfile.ValueFloat(*bvalue)

	case *ValueDouble:
		value = rbxfile.ValueDouble(*bvalue)

	case *ValueUDim:
		value = rbxfile.ValueUDim{
			Scale:  float32(bvalue.Scale),
			Offset: int32(bvalue.Offset),
		}

	case *ValueUDim2:
		value = rbxfile.ValueUDim2{
			X: rbxfile.ValueUDim{
				Scale:  float32(bvalue.ScaleX),
				Offset: int32(bvalue.OffsetX),
			},
			Y: rbxfile.ValueUDim{
				Scale:  float32(bvalue.ScaleY),
				Offset: int32(bvalue.OffsetY),
			},
		}

	case *ValueRay:
		value = rbxfile.ValueRay{
			Origin: rbxfile.ValueVector3{
				X: bvalue.OriginX,
				Y: bvalue.OriginY,
				Z: bvalue.OriginZ,
			},
			Direction: rbxfile.ValueVector3{
				X: bvalue.DirectionX,
				Y: bvalue.DirectionY,
				Z: bvalue.DirectionZ,
			},
		}

	case *ValueFaces:
		value = rbxfile.ValueFaces(*bvalue)

	case *ValueAxes:
		value = rbxfile.ValueAxes(*bvalue)

	case *ValueBrickColor:
		value = rbxfile.ValueBrickColor(*bvalue)

	case *ValueColor3:
		value = rbxfile.ValueColor3{
			R: float32(bvalue.R),
			G: float32(bvalue.G),
			B: float32(bvalue.B),
		}

	case *ValueVector2:
		value = rbxfile.ValueVector2{
			X: float32(bvalue.X),
			Y: float32(bvalue.Y),
		}

	case *ValueVector3:
		value = rbxfile.ValueVector3{
			X: float32(bvalue.X),
			Y: float32(bvalue.Y),
			Z: float32(bvalue.Z),
		}

	case *ValueVector2int16:
		value = rbxfile.ValueVector2int16(*bvalue)

	case *ValueCFrame:
		cf := rbxfile.ValueCFrame{
			Position: rbxfile.ValueVector3{
				X: float32(bvalue.Position.X),
				Y: float32(bvalue.Position.Y),
				Z: float32(bvalue.Position.Z),
			},
			Rotation: bvalue.Rotation,
		}

		if bvalue.Special != 0 {
			cf.Rotation = cframeSpecialMatrix[bvalue.Special]
		}

		value = cf

	case *ValueToken:
		value = rbxfile.ValueToken(*bvalue)

	case *ValueReference:
		// Must be resolved elsewhere.

	case *ValueVector3int16:
		value = rbxfile.ValueVector3int16(*bvalue)

	case *ValueNumberSequence:
		v := make(rbxfile.ValueNumberSequence, len(*bvalue))
		for i, nsk := range *bvalue {
			v[i] = rbxfile.ValueNumberSequenceKeypoint{
				Time:     nsk.Time,
				Value:    nsk.Value,
				Envelope: nsk.Envelope,
			}
		}
		value = v

	case *ValueColorSequence:
		v := make(rbxfile.ValueColorSequence, len(*bvalue))
		for i, nsk := range *bvalue {
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
		value = v

	case *ValueNumberRange:
		value = rbxfile.ValueNumberRange(*bvalue)

	case *ValueRect2D:
		value = rbxfile.ValueRect2D{
			Min: rbxfile.ValueVector2{
				X: float32(bvalue.Min.X),
				Y: float32(bvalue.Min.Y),
			},
			Max: rbxfile.ValueVector2{
				X: float32(bvalue.Max.X),
				Y: float32(bvalue.Max.Y),
			},
		}

	case *ValuePhysicalProperties:
		value = rbxfile.ValuePhysicalProperties{
			CustomPhysics:    bvalue.CustomPhysics != 0,
			Density:          bvalue.Density,
			Friction:         bvalue.Friction,
			Elasticity:       bvalue.Elasticity,
			FrictionWeight:   bvalue.FrictionWeight,
			ElasticityWeight: bvalue.ElasticityWeight,
		}

	case *ValueColor3uint8:
		value = rbxfile.ValueColor3uint8{
			R: bvalue.R,
			G: bvalue.G,
			B: bvalue.B,
		}

	case *ValueInt64:
		value = rbxfile.ValueInt64(*bvalue)

	case *ValueSharedString:
		// Must be resolved elsewhere.
	}

	return
}

type sharedEntry struct {
	index int
	value SharedString
}

type sharedMap map[[16]byte]sharedEntry

func (c RobloxCodec) Encode(root *rbxfile.Root) (model *FormatModel, err error) {
	if root == nil {
		return nil, errors.New("Root is nil")
	}

	model = new(FormatModel)

	// A list of instances in the tree. The index serves as the instance's
	// reference number.
	instList := make([]*rbxfile.Instance, 0)

	// A map used to ensure that an instance is counted only once. Also used
	// to link ValueReferences.
	refs := map[*rbxfile.Instance]int{
		// In general, -1 indicates a nil value.
		nil: -1,
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
	instChunkMap := map[string]*ChunkInstance{}
	for ref, inst := range instList {

		chunk, ok := instChunkMap[inst.ClassName]
		if !ok {
			chunk = &ChunkInstance{
				IsCompressed: true,
				ClassName:    inst.ClassName,
				InstanceIDs:  []int32{},
			}
			instChunkMap[inst.ClassName] = chunk
		}

		chunk.InstanceIDs = append(chunk.InstanceIDs, int32(ref))

		if c.Mode == ModePlace && inst.IsService {
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
	propChunkList := []*ChunkProperty{}
	for i, instChunk := range instChunkList {
		instChunk.ClassID = int32(i)

		addWarn := func(format string, v ...interface{}) {
			q := make([]interface{}, 0, len(v)+1)
			q = append(q, instChunk.ClassID)
			q = append(q, v...)
			model.Warnings = append(model.Warnings, fmt.Errorf("instance chunk #%d: "+format, q...))
		}

		propChunkMap := map[string]*ChunkProperty{}

		// Populate propChunkMap.
		for _, ref := range instChunk.InstanceIDs {
			for name := range instList[ref].Properties {
				if _, ok := propChunkMap[name]; ok {
					// A chunk of the property name already exists.
					continue
				}
				propChunkMap[name] = &ChunkProperty{
					IsCompressed: true,
					ClassID:      instChunk.ClassID,
					PropertyName: name,
					Properties:   make([]Value, len(instChunk.InstanceIDs)),
				}
			}
		}

		// Check to see if all existing properties types match.
		for name, propChunk := range propChunkMap {
			var instRef int32 = -1
			dataType := rbxfile.TypeInvalid
			matches := true
			for _, ref := range instChunk.InstanceIDs {
				inst := instList[ref]
				prop, ok := inst.Properties[name]
				if !ok {
					continue
				}
				if dataType == rbxfile.TypeInvalid {
					// Set data type to the first valid property.
					dataType = prop.Type()
					instRef = ref
					continue
				}
				if prop.Type() != dataType {
					// If at least one property type does not match with the
					// rest, then stop.
					matches = false
					break
				}
			}
			if !matches {
				delete(propChunkMap, name)
				addWarn("mismatched types for property %s.%s, chunk skipped", instList[instRef].ClassName, name)
				continue
			}
			btype := FromValueType(dataType)
			if btype == TypeInvalid {
				delete(propChunkMap, name)
				addWarn("unknown type %d for property %s.%s in instance #%d, chunk skipped", byte(dataType), instList[instRef].ClassName, name, instRef)
				continue
			}
			propChunk.DataType = btype
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
							ref = -1
						}

						v := int32(ref)
						bvalue = (*ValueReference)(&v)
					case rbxfile.ValueSharedString:
						// TODO: verify that strings are compared by MD5 hash.
						hash := md5.Sum([]byte(value))
						entry, ok := sharedStrings[hash]
						if !ok {
							entry.index = len(sharedStrings)
							entry.value.Hash = hash
							entry.value.Value = []byte(value)
							sharedStrings[hash] = entry
						}
						index := uint32(entry.index)
						bvalue = (*ValueSharedString)(&index)
					default:
						bvalue = encodeValue(value)
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
	parentChunk := &ChunkParent{
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
				parentRef = -1
			}
			parentChunk.Parents[i] = int32(parentRef)
			i++
		}
		for _, inst := range root.Instances {
			recInst(inst, nil)
		}
	}

	// Make end chunk.
	endChunk := &ChunkEnd{
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
	model.Chunks = make([]Chunk, 0, chunkLength)

	if len(root.Metadata) > 0 {
		// TODO: verify that chunk is omitted when zero values are encoded, and
		// is not based on format (RBXM vs RBXL).
		chunk := ChunkMeta{
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
		chunk := ChunkSharedStrings{
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

type sortInstChunks []*ChunkInstance

func (c sortInstChunks) Len() int {
	return len(c)
}
func (c sortInstChunks) Less(i, j int) bool {
	return c[i].ClassName < c[j].ClassName
}
func (c sortInstChunks) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type sortPropChunks []*ChunkProperty

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

func encodeValue(value rbxfile.Value) (bvalue Value) {
	switch value := value.(type) {
	case rbxfile.ValueString:
		v := make([]byte, len(value))
		copy(v, value)
		bvalue = (*ValueString)(&v)

	case rbxfile.ValueBinaryString:
		v := make([]byte, len(value))
		copy(v, value)
		bvalue = (*ValueString)(&v)

	case rbxfile.ValueProtectedString:
		v := make([]byte, len(value))
		copy(v, value)
		bvalue = (*ValueString)(&v)

	case rbxfile.ValueContent:
		v := make([]byte, len(value))
		copy(v, value)
		bvalue = (*ValueString)(&v)

	case rbxfile.ValueBool:
		bvalue = (*ValueBool)(&value)

	case rbxfile.ValueInt:
		bvalue = (*ValueInt)(&value)

	case rbxfile.ValueFloat:
		bvalue = (*ValueFloat)(&value)

	case rbxfile.ValueDouble:
		bvalue = (*ValueDouble)(&value)

	case rbxfile.ValueUDim:
		bvalue = &ValueUDim{
			Scale:  ValueFloat(value.Scale),
			Offset: ValueInt(value.Offset),
		}

	case rbxfile.ValueUDim2:
		bvalue = &ValueUDim2{
			ScaleX:  ValueFloat(value.X.Scale),
			ScaleY:  ValueFloat(value.Y.Scale),
			OffsetX: ValueInt(value.X.Offset),
			OffsetY: ValueInt(value.Y.Offset),
		}

	case rbxfile.ValueRay:
		bvalue = &ValueRay{
			OriginX:    value.Origin.X,
			OriginY:    value.Origin.Y,
			OriginZ:    value.Origin.Z,
			DirectionX: value.Direction.X,
			DirectionY: value.Direction.Y,
			DirectionZ: value.Direction.Z,
		}

	case rbxfile.ValueFaces:
		bvalue = (*ValueFaces)(&value)

	case rbxfile.ValueAxes:
		bvalue = (*ValueAxes)(&value)

	case rbxfile.ValueBrickColor:
		bvalue = (*ValueBrickColor)(&value)

	case rbxfile.ValueColor3:
		bvalue = &ValueColor3{
			R: ValueFloat(value.R),
			G: ValueFloat(value.G),
			B: ValueFloat(value.B),
		}

	case rbxfile.ValueVector2:
		bvalue = &ValueVector2{
			X: ValueFloat(value.X),
			Y: ValueFloat(value.Y),
		}

	case rbxfile.ValueVector3:
		bvalue = &ValueVector3{
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

		bvalue = cf

	case rbxfile.ValueToken:
		return (*ValueToken)(&value)

	case rbxfile.ValueReference:
		// Must be resolved elsewhere.

	case rbxfile.ValueVector3int16:
		bvalue = (*ValueVector3int16)(&value)

	case rbxfile.ValueVector2int16:
		bvalue = (*ValueVector2int16)(&value)

	case rbxfile.ValueNumberSequence:
		v := make(ValueNumberSequence, len(value))
		for i, nsk := range value {
			v[i] = ValueNumberSequenceKeypoint{
				Time:     nsk.Time,
				Value:    nsk.Value,
				Envelope: nsk.Envelope,
			}
		}
		bvalue = &v

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
		bvalue = &v

	case rbxfile.ValueNumberRange:
		bvalue = (*ValueNumberRange)(&value)

	case rbxfile.ValueRect2D:
		bvalue = &ValueRect2D{
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
		bvalue = &v

	case rbxfile.ValueColor3uint8:
		bvalue = &ValueColor3uint8{
			R: value.R,
			G: value.G,
			B: value.B,
		}

	case rbxfile.ValueInt64:
		bvalue = (*ValueInt64)(&value)

	case rbxfile.ValueSharedString:
		// Must be resolved elsewhere.
	}

	return
}
