package bin

import (
	"errors"
	"fmt"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxfile"
	"sort"
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
	// API can be set to yield a more correct encoding or decoding by
	// providing information about each class. If API is nil, the codec will
	// try to use other available information, but may not be fully accurate.
	API rbxapi.Root

	Mode Mode

	// ExcludeInvalidAPI determines whether invalid items are excluded when
	// encoding or decoding. An invalid item is an instance or property that
	// does not exist or has incorrect information, according to a provided
	// rbxapi.Root.
	//
	// If true, then warnings will be emitted for invalid items, and the items
	// will not be included in the output. If false, then warnings are still
	// emitted, but invalid items are handled as if they were valid. This
	// applies when decoding from a FormatModel, and when encoding from a
	// rbxfile.Root.
	//
	// Since an API may exclude some items even though they're correct, it is
	// generally preferred to set ExcludeInvalidAPI to false, so that false
	// negatives do not lead to lost data.
	ExcludeInvalidAPI bool
}

//go:generate rbxpipe -i=cframegen.lua -o=cframe.go -place=cframe.rbxl -filter=o

func (c RobloxCodec) Decode(model *FormatModel) (root *rbxfile.Root, err error) {
	if model == nil {
		return nil, fmt.Errorf("FormatModel is nil")
	}
	model.Warnings = model.Warnings[:0]

	root = new(rbxfile.Root)

	groupLookup := make(map[int32]*ChunkInstance, model.TypeCount)
	instLookup := make(map[int32]*rbxfile.Instance, model.InstanceCount+1)
	instLookup[-1] = nil

	propTypes := map[string]map[string]rbxapi.Type{}

	// Caches an enum name to a set of enum item values.
	enumCache := map[string]enumItems{}

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
			if chunk.TypeID < 0 || uint32(chunk.TypeID) >= model.TypeCount {
				err = fmt.Errorf("type index out of bounds: %d", model.TypeCount)
				goto chunkErr
			}
			// No error if TypeCount > actual count.

			if c.API != nil {
				class := c.API.GetClass(chunk.ClassName)
				if class == nil {
					// Invalid ClassNames cause the chunk to be ignored.
					addWarn("invalid ClassName `%s`", chunk.ClassName)
					if c.ExcludeInvalidAPI {
						continue
					}
				}

				// Cache property names and types for the class.
				if _, ok := propTypes[chunk.ClassName]; !ok {
					props := map[string]rbxapi.Type{}
					for _, member := range class.GetMembers() {
						if member, ok := member.(rbxapi.Property); ok {
							props[member.GetName()] = member.GetValueType()

							// Check if property type is an enum.
							enum := c.API.GetEnum(member.GetValueType().GetName())
							if enum == nil {
								continue
							}

							// Generate an enum items map to be used later.
							items, ok := enumCache[member.GetValueType().GetName()]
							if !ok {
								itemList := enum.GetEnumItems()
								items = enumItems{
									first:  itemList[0].GetValue(),
									values: make(map[int]bool, len(itemList)),
								}
								for _, item := range itemList {
									items.values[item.GetValue()] = true
								}
								enumCache[member.GetValueType().GetName()] = items
							}
						}
					}
					propTypes[chunk.ClassName] = props
				}
			}

			if chunk.IsService && len(chunk.InstanceIDs) != len(chunk.GetService) {
				err = fmt.Errorf("malformed instance chunk (type ID %d): GetService array length does not equal InstanceIDs array length", chunk.TypeID)
				goto chunkErr
			}

			for i, ref := range chunk.InstanceIDs {
				if ref < 0 || uint32(ref) >= model.InstanceCount {
					err = fmt.Errorf("invalid id %d", ref)
					goto chunkErr
				}
				// No error if InstanceCount > actual count.

				inst := rbxfile.NewInstance(chunk.ClassName, nil)
				if _, ok := instLookup[ref]; ok {
					err = fmt.Errorf("duplicate id: %d", ref)
					goto chunkErr
				}

				if chunk.IsService && chunk.GetService[i] == 1 {
					inst.IsService = true
				}

				instLookup[ref] = inst
			}

			if _, ok := groupLookup[chunk.TypeID]; ok {
				err = fmt.Errorf("duplicate type index: %d", chunk.TypeID)
				goto chunkErr
			}
			groupLookup[chunk.TypeID] = chunk

		case *ChunkProperty:
			chunkType = "property"
			if chunk.TypeID < 0 || uint32(chunk.TypeID) >= model.TypeCount {
				err = fmt.Errorf("type index out of bounds: %d", model.TypeCount)
				goto chunkErr
			}
			// No error if TypeCount > actual count.

			instChunk, ok := groupLookup[chunk.TypeID]
			if !ok {
				addWarn("type `%d` of property group is invalid or unknown", chunk.TypeID)
				continue
			}

			if len(chunk.Properties) != len(instChunk.InstanceIDs) {
				err = fmt.Errorf("length of properties array (%d) does not equal length of type array (%d)", len(chunk.Properties), len(instChunk.InstanceIDs))
				goto chunkErr
			}

			var propType rbxapi.Type
			if c.API != nil {
				var ok bool
				if propType, ok = propTypes[instChunk.ClassName][chunk.PropertyName]; !ok {
					addWarn("chunk name `%s` is not a valid property of the group class `%s`", chunk.PropertyName, instChunk.ClassName)
					if c.ExcludeInvalidAPI {
						continue
					}
				}
			}

			for i, bvalue := range chunk.Properties {
				// If the value type is an enum, then verify that the value is
				// correct for the enum.
				if c.API != nil && bvalue.Type() == TypeToken {
					if items, ok := enumCache[propType.GetName()]; ok {
						token := bvalue.(*ValueToken)
						if !items.values[int(*token)] {
							// If it isn't valid, then use the first value of
							// the enum instead.
							v := items.first
							*token = ValueToken(v)
						}
					}
				}

				inst := instLookup[instChunk.InstanceIDs[i]]
				inst.Properties[chunk.PropertyName] = decodeValue(propType, instLookup, bvalue)
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

				if err = parent.AddChild(child); err != nil {
					goto chunkErr
				}

			}

		case *ChunkMeta:
			chunkType = "meta"
			if root.Metadata == nil {
				root.Metadata = make(map[string]string, len(chunk.Values))
			}
			for _, pair := range chunk.Values {
				root.Metadata[pair[0]] = pair[1]
			}

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

// Decode a bin.value to a rbxfile.Value based on a given value type.
func decodeValue(valueType rbxapi.Type, refs map[int32]*rbxfile.Instance, bvalue Value) (value rbxfile.Value) {
	switch bvalue := bvalue.(type) {
	case *ValueString:
		v := make([]byte, len(*bvalue))
		copy(v, *bvalue)

		if valueType == nil {
			value = rbxfile.ValueString(v)
			break
		}
		switch valueType.GetName() {
		case rbxfile.TypeBinaryString.String():
			value = rbxfile.ValueBinaryString(v)

		case rbxfile.TypeProtectedString.String():
			value = rbxfile.ValueProtectedString(v)

		case rbxfile.TypeContent.String():
			value = rbxfile.ValueContent(v)

		case rbxfile.TypeString.String():
			fallthrough

		default:
			value = rbxfile.ValueString(v)
		}

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
			Offset: int16(bvalue.Offset),
		}

	case *ValueUDim2:
		value = rbxfile.ValueUDim2{
			X: rbxfile.ValueUDim{
				Scale:  float32(bvalue.ScaleX),
				Offset: int16(bvalue.OffsetX),
			},
			Y: rbxfile.ValueUDim{
				Scale:  float32(bvalue.ScaleY),
				Offset: int16(bvalue.OffsetY),
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
		value = rbxfile.ValueReference{Instance: refs[int32(*bvalue)]}

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

	}

	return
}

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

	// Recursively finds and adds instances.
	var addInstance func(inst *rbxfile.Instance)
	addInstance = func(inst *rbxfile.Instance) {
		if _, ok := refs[inst]; ok {
			// Ignore the instance if it has already been read.
			return
		}

		if c.API != nil {
			if class := c.API.GetClass(inst.ClassName); class == nil {
				model.Warnings = append(model.Warnings, fmt.Errorf("invalid ClassName `%s`", inst.ClassName))
				if c.ExcludeInvalidAPI {
					return
				}
			}
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
		typeID := 0
		for _, chunk := range instChunkMap {
			instChunkList[typeID] = chunk
			typeID++
		}

		sort.Sort(instChunkList)
	}

	// Caches an enum name to a set of enum item values.
	enumCache := map[string]enumItems{}

	// Make property chunks.
	propChunkList := []*ChunkProperty{}
	for i, instChunk := range instChunkList {
		instChunk.TypeID = int32(i)

		addWarn := func(format string, v ...interface{}) {
			q := make([]interface{}, 0, len(v)+1)
			q = append(q, instChunk.TypeID)
			q = append(q, v...)
			model.Warnings = append(model.Warnings, fmt.Errorf("instance chunk #%d: "+format, q...))
		}

		// Maps property names to enum items.
		propEnums := map[string]enumItems{}

		propChunkMap := map[string]*ChunkProperty{}

		var propAPI map[string]rbxapi.Property
		if c.API != nil {
			propAPI = make(map[string]rbxapi.Property)

			// Should exist due to previous checks.
			class := c.API.GetClass(instChunk.ClassName)

			for _, member := range class.GetMembers() {
				member, ok := member.(rbxapi.Property)
				if !ok {
					continue
				}
				propAPI[member.GetName()] = member
			}
		}

		// Populate propChunkMap.
		for _, ref := range instChunk.InstanceIDs {
			inst := instList[ref]
			for name, value := range inst.Properties {
				if _, ok := propChunkMap[name]; ok {
					// A chunk of the property name already exists.
					continue
				}

				dataType := TypeInvalid
				if propAPI != nil {
					member, ok := propAPI[name]
					if !ok {
						addWarn("invalid property %s.`%s`", inst.ClassName, name)
						if c.ExcludeInvalidAPI {
							// Skip over the property entirely.
							continue
						}
						// Fallback to the given property type as the chunk
						// type.
						goto useFirst
					}
					typ := rbxfile.TypeFromString(member.GetValueType().GetName())
					if typ == rbxfile.TypeInvalid {
						// Check if property type is an enum.
						enum := c.API.GetEnum(member.GetValueType().GetName())
						if enum == nil {
							addWarn("encountered unknown data type `%s` in API", member.GetValueType())
							if c.ExcludeInvalidAPI {
								continue
							}
							goto useFirst
						}

						// TypeToken represents an enum.
						typ = rbxfile.TypeToken

						// Generate an enum items map to be used later.
						items, ok := enumCache[enum.GetName()]
						if !ok {
							itemList := enum.GetEnumItems()
							items = enumItems{
								name:   enum.GetName(),
								first:  itemList[0].GetValue(),
								values: make(map[int]bool, len(itemList)),
							}
							for _, item := range itemList {
								items.values[item.GetValue()] = true
							}
							enumCache[enum.GetName()] = items
						}
						propEnums[member.GetName()] = items
					}

					bval := encodeValue(refs, rbxfile.NewValue(typ))
					if bval == nil {
						addWarn("encountered unknown data type `%s` in API", member.GetValueType())
						if c.ExcludeInvalidAPI {
							continue
						}
						goto useFirst
					}
					// Use API type as property chunk type.
					dataType = bval.Type()
					goto finish
				}

			useFirst:
				// Use type from existing property in the first instance.
				{
					bval := encodeValue(refs, value)
					if bval == nil {
						addWarn("unknown property type (%d) in instance #%d (%s.%s)", byte(value.Type()), ref, inst.ClassName, name)
						continue
					}
					dataType = bval.Type()
				}

			finish:
				propChunkMap[name] = &ChunkProperty{
					IsCompressed: true,
					TypeID:       instChunk.TypeID,
					PropertyName: name,
					DataType:     dataType,
					Properties:   make([]Value, len(instChunk.InstanceIDs)),
				}
			}
		}

		if propAPI != nil && !c.ExcludeInvalidAPI {
			// Check to see if all existing properties types match. If they
			// do, prefer those types over the API's type.
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

				if matches {
					bval := encodeValue(refs, rbxfile.NewValue(dataType))
					if bval == nil {
						addWarn("unknown property data type (%d) in instance #%d (%s.%s)", byte(dataType), instRef, instList[instRef].ClassName, name)
						continue
					}
					propChunk.DataType = bval.Type()
				}
			}
		}

		// Set the values for each property chunk.
		for name, propChunk := range propChunkMap {
			for i, ref := range instChunk.InstanceIDs {
				inst := instList[ref]

				var bvalue Value
				if value, ok := inst.Properties[name]; ok {
					bvalue = encodeValue(refs, value)
				}

				if bvalue == nil || bvalue.Type() != propChunk.DataType {
					// Use default value for DataType.
					bvalue = NewValue(propChunk.DataType)
				}

				// If the value type is an enum, then verify that the value is
				// correct for the enum.
				if c.API != nil && bvalue.Type() == TypeToken {
					if items, ok := propEnums[name]; ok {
						token := bvalue.(*ValueToken)
						if !items.values[int(*token)] {
							addWarn("invalid value `%d` for enum %s in instance #%d (%s.%s)", token, items.name, ref, inst.ClassName, name)
							if c.ExcludeInvalidAPI {
								// If it isn't valid, then use the first value of
								// the enum instead.
								v := items.first
								*token = ValueToken(v)
							}
						}
					}
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
		var recInst func(inst *rbxfile.Instance)
		recInst = func(inst *rbxfile.Instance) {
			for _, child := range inst.Children {
				recInst(child)
			}

			instRef := int32(refs[inst])
			parentChunk.Children[i] = instRef
			parentRef, ok := refs[inst.Parent()]
			if !ok {
				parentRef = -1
			}
			parentChunk.Parents[i] = int32(parentRef)
			i++
		}
		for _, inst := range root.Instances {
			recInst(inst)
		}
	}

	// Make end chunk.
	endChunk := &ChunkEnd{
		IsCompressed: false,
		Content:      []byte("</roblox>"),
	}

	// Make FormatModel.
	model.TypeCount = uint32(len(instChunkList))
	model.InstanceCount = uint32(len(instList))
	model.Chunks = make([]Chunk, 1+len(instChunkList)+len(propChunkList)+1+1)
	chunks := model.Chunks[:0]

	metaChunk := &ChunkMeta{
		IsCompressed: true,
		Values:       make([][2]string, 0, len(root.Metadata)),
	}
	for key, value := range root.Metadata {
		metaChunk.Values = append(metaChunk.Values, [2]string{key, value})
	}
	sort.Sort(sortMetaData(metaChunk.Values))
	chunks = chunks[len(chunks) : len(chunks)+1]
	chunks[0] = metaChunk

	chunks = chunks[len(chunks) : len(chunks)+len(instChunkList)]
	for i, chunk := range instChunkList {
		chunks[i] = chunk
	}

	chunks = chunks[len(chunks) : len(chunks)+len(propChunkList)]
	for i, chunk := range propChunkList {
		chunks[i] = chunk
	}

	chunks = chunks[len(chunks) : len(chunks)+1]
	chunks[0] = parentChunk

	chunks = chunks[len(chunks) : len(chunks)+1]
	chunks[0] = endChunk

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

type enumItems struct {
	name   string
	first  int
	values map[int]bool
}

func encodeValue(refs map[*rbxfile.Instance]int, value rbxfile.Value) (bvalue Value) {
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
		// Convert an instance reference to a reference number.
		ref, ok := refs[value.Instance]
		if !ok {
			// References that map to some instance not under the
			// Root should be nil.
			ref = -1
		}

		v := int32(ref)
		bvalue = (*ValueReference)(&v)

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
	}

	return
}
