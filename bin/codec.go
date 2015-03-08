package bin

import (
	"fmt"
	"github.com/robloxapi/rbxdump"
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
	Mode Mode
}

//go:generate rbxpipe -i=cframegen.lua -o=cframe.go -place=cframe.rbxl -filter=o

func (c RobloxCodec) Decode(model *FormatModel, api *rbxdump.API) (root *rbxfile.Root, err error) {
	root = new(rbxfile.Root)

	groupLookup := make(map[int32]*ChunkInstance, model.TypeCount)
	instLookup := make(map[int32]*rbxfile.Instance, model.InstanceCount+1)
	instLookup[-1] = nil

	propTypes := map[string]map[string]string{}

	// Caches an enum name to a set of enum item values.
	enumCache := map[string]enumItems{}

loop:
	for ic, chunk := range model.Chunks {
		switch chunk := chunk.(type) {
		case *ChunkInstance:
			if chunk.TypeID < 0 || uint32(chunk.TypeID) >= model.TypeCount {
				return nil, fmt.Errorf("type index out of bounds: %d", model.TypeCount)
			}
			// No error if TypeCount > actual count.

			if api != nil {
				class, ok := api.Classes[chunk.ClassName]
				if !ok {
					// Invalid ClassNames cause the chunk to be ignored.
					// WARNING: invalid ClassName
					continue
				}

				// Cache property names and types for the class.
				if _, ok := propTypes[chunk.ClassName]; !ok {
					props := map[string]string{}
					for _, member := range class.Members {
						if member, ok := member.(*rbxdump.Property); ok {
							props[member.Name] = member.ValueType

							// Check if property type is an enum.
							enum, ok := api.Enums[member.ValueType]
							if !ok {
								continue
							}

							// Generate an enum items map to be used later.
							items, ok := enumCache[member.ValueType]
							if !ok {
								items = enumItems{
									first:  enum.Items[0].Value,
									values: make(map[uint32]bool, len(enum.Items)),
								}
								for _, item := range enum.Items {
									items.values[item.Value] = true
								}
								enumCache[member.ValueType] = items
							}
						}
					}
					propTypes[chunk.ClassName] = props
				}
			}

			if chunk.IsService && len(chunk.InstanceIDs) != len(chunk.GetService) {
				return nil, fmt.Errorf("malformed instance chunk (type ID %d): GetService array length does not equal InstanceIDs array length", chunk.TypeID)
			}

			for i, ref := range chunk.InstanceIDs {
				if ref < 0 || uint32(ref) >= model.InstanceCount {
					return nil, fmt.Errorf("invalid id %d", ref)
				}
				// No error if InstanceCount > actual count.

				inst := rbxfile.NewInstance(chunk.ClassName, nil)
				if _, ok := instLookup[ref]; ok {
					return nil, fmt.Errorf("duplicate id: %d", ref)
				}

				if chunk.IsService && chunk.GetService[i] == 1 {
					inst.IsService = true
				}

				instLookup[ref] = inst
			}

			if _, ok := groupLookup[chunk.TypeID]; ok {
				return nil, fmt.Errorf("duplicate type index: %d", chunk.TypeID)
			}
			groupLookup[chunk.TypeID] = chunk

		case *ChunkProperty:
			if chunk.TypeID < 0 || uint32(chunk.TypeID) >= model.TypeCount {
				return nil, fmt.Errorf("type index out of bounds: %d", model.TypeCount)
			}
			// No error if TypeCount > actual count.

			instChunk, ok := groupLookup[chunk.TypeID]
			if !ok {
				// WARNING: type of property group is invalid or unknown
				continue
			}

			if len(chunk.Properties) != len(instChunk.InstanceIDs) {
				return nil, fmt.Errorf("length of properties array (%d) does not equal length of type array (%d)", len(chunk.Properties), len(instChunk.InstanceIDs))
			}

			var propType string
			if api != nil {
				var ok bool
				if propType, ok = propTypes[instChunk.ClassName][chunk.PropertyName]; !ok {
					// WARNING: chunk name is not a valid property of the group class
					continue
				}
			}

			for i, bvalue := range chunk.Properties {
				// If the value type is an enum, then verify that the value is
				// correct for the enum.
				if api != nil && bvalue.Type() == TypeToken {
					if items, ok := enumCache[propType]; ok {
						token := bvalue.(*ValueToken)
						if !items.values[uint32(*token)] {
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
			if chunk.Version != 0 {
				return nil, fmt.Errorf("unrecognized parent link format %d", chunk.Version)
			}

			if len(chunk.Parents) != len(chunk.Children) {
				return nil, fmt.Errorf("malformed parent chunk (#%d): length of Parents array does not equal length of Children array", ic)
			}

			for i, ref := range chunk.Children {
				if ref < 0 || uint32(ref) >= model.InstanceCount {
					return nil, fmt.Errorf("invalid id %d", ref)
				}

				child := instLookup[ref]
				if child == nil {
					// WARNING: referent does not exist
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

				if err := child.SetParent(parent); err != nil {
					return nil, err
				}

			}

		case *ChunkEnd:
			break loop
		}
	}

	return
}

// Decode a bin.value to a rbxfile.Value based on a given value type.
func decodeValue(valueType string, refs map[int32]*rbxfile.Instance, bvalue Value) (value rbxfile.Value) {
	switch bvalue := bvalue.(type) {
	case *ValueString:
		v := make([]byte, len(*bvalue))
		copy(v, *bvalue)

		switch valueType {
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
		value = rbxfile.ValueReference{Instance: refs[int32(*bvalue)]}

	case *ValueVector3int16:
		value = rbxfile.ValueVector3int16(*bvalue)
	}

	return
}

func (c RobloxCodec) Encode(root *rbxfile.Root, api *rbxdump.API) (model *FormatModel, err error) {
	model = NewFormatModel()

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

		if api != nil {
			if _, ok := api.Classes[inst.ClassName]; !ok {
				// Warning: Instance `` does not have a valid ClassName.
				return
			}
		}

		// Reference number should match position in list.
		refs[inst] = len(instList)
		instList = append(instList, inst)

		for _, child := range inst.GetChildren() {
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
			inst.IsService = true
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

		// Maps property names to enum items.
		propEnums := map[string]enumItems{}

		propChunkMap := map[string]*ChunkProperty{}

		// Populate propChunkMap.
		if api != nil {
			// Use members of the current instance.

			// Should exist due to previous checks.
			class := api.Classes[instChunk.ClassName]

			for _, member := range class.Members {
				member, ok := member.(*rbxdump.Property)
				if !ok {
					continue
				}

				// Read-only members should not be written.
				if member.ReadOnly {
					continue
				}

				typ := rbxfile.TypeFromString(member.ValueType)
				if typ == rbxfile.TypeInvalid {
					// Check if property type is an enum.
					enum, ok := api.Enums[member.ValueType]
					if !ok {
						// Property type is not known; ignore it.
						continue
					}

					// TypeToken represents an enum.
					typ = rbxfile.TypeToken

					// Generate an enum items map to be used later.
					items, ok := enumCache[enum.Name]
					if !ok {
						items = enumItems{
							first:  enum.Items[0].Value,
							values: make(map[uint32]bool, len(enum.Items)),
						}
						for _, item := range enum.Items {
							items.values[item.Value] = true
						}
						enumCache[enum.Name] = items
					}
					propEnums[member.Name] = items
				}

				bval := encodeValue(refs, rbxfile.NewValue(typ))
				if bval == nil {
					// Property type cannot be encoded; ignore it.
					continue
				}

				propChunkMap[member.Name] = &ChunkProperty{
					IsCompressed: true,
					TypeID:       instChunk.TypeID,
					PropertyName: member.Name,
					DataType:     bval.Type(),
					Properties:   []Value{},
				}
			}
		} else {
			// Assume that each instance's properties are valid.
			for _, ref := range instChunk.InstanceIDs {
				for name, value := range instList[ref].Properties {
					// The DataType of the chunk is selected by the first
					// instance that has a valid value of the property.

					if _, ok := propChunkMap[name]; ok {
						// A chunk of the property name already exists.
						continue
					}

					if value.Type() == rbxfile.TypeInvalid {
						// Property type is not known; ignore it.
						continue
					}

					bval := encodeValue(refs, value)
					if bval == nil {
						// Property type cannot be encoded; ignore it.
						continue
					}

					propChunkMap[name] = &ChunkProperty{
						IsCompressed: true,
						TypeID:       instChunk.TypeID,
						PropertyName: name,
						DataType:     bval.Type(),
						Properties:   []Value{},
					}
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
				if api != nil && bvalue.Type() == TypeToken {
					if items, ok := propEnums[name]; ok {
						token := bvalue.(*ValueToken)
						if !items.values[uint32(*token)] {
							// If it isn't valid, then use the first value of
							// the enum instead.
							v := items.first
							*token = ValueToken(v)
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
		for _, instChunk := range instChunkList {
			for _, instRef := range instChunk.InstanceIDs {
				parentChunk.Children[i] = instRef
				inst := instList[instRef]
				parentRef, ok := refs[inst.Parent()]
				if !ok {
					parentRef = -1
				}
				parentChunk.Parents[i] = int32(parentRef)
				i++
			}
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
	model.Chunks = make([]Chunk, len(instChunkList)+len(propChunkList)+1+1)
	chunks := model.Chunks[:0]

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

type enumItems struct {
	first  uint32
	values map[uint32]bool
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
	}

	return
}
