package rbxlx

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/robloxapi/rbxfile"
	"golang.org/x/crypto/blake2b"
)

// robloxCodec implements Decoder and Encoder to emulate Roblox's internal
// codec as closely as possible.
type robloxCodec struct {
	// ExcludeReferent determines whether the "referent" attribute should be
	// added to Item tags when encoding.
	ExcludeReferent bool

	// ExcludeExternal determines whether standard <External> tags should be
	// added to the root tag when encoding.
	ExcludeExternal bool

	// ExcludeMetadata determines whether <Meta> tags should be included while
	// encoding.
	ExcludeMetadata bool

	// DiscardInvalidProperties determines how invalid properties are decoded.
	// If true, when the parser successfully decodes a property, but fails to
	// decode its value or a component, then the entire property is discarded.
	// If false, then as much information as possible is retained; any value or
	// component that fails will be emitted as the zero value for the type.
	DiscardInvalidProperties bool
}

func (c robloxCodec) Decode(document *documentRoot) (root *rbxfile.Root, err error) {
	if document == nil {
		return nil, fmt.Errorf("document is nil")
	}

	dec := &rdecoder{
		document:   document,
		codec:      c,
		root:       new(rbxfile.Root),
		instLookup: make(rbxfile.References),
	}

	dec.decode()
	return dec.root, dec.err
}

type rdecoder struct {
	document   *documentRoot
	codec      robloxCodec
	root       *rbxfile.Root
	err        error
	instLookup rbxfile.References
	propRefs   []rbxfile.PropRef
	stringRefs []rbxfile.PropRef
}

func (dec *rdecoder) decode() error {
	if dec.err != nil {
		return dec.err
	}
	if dec.document.Root == nil {
		return errors.New("no root tag")
	}

	dec.root = new(rbxfile.Root)
	dec.root.Instances, _ = dec.getItems(nil, dec.document.Root.Tags)

	for _, tag := range dec.document.Root.Tags {
		switch tag.StartName {
		case "Meta":
			key, ok := tag.AttrValue("name")
			if !ok {
				continue
			}
			if dec.root.Metadata == nil {
				dec.root.Metadata = make(map[string]string)
			}
			dec.root.Metadata[key] = tag.Text
		case "SharedStrings":
			for _, tag := range tag.Tags {
				if tag.StartName != "SharedString" {
					continue
				}
				hash, ok := tag.AttrValue("md5")
				if !ok {
					continue
				}
				key, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(hash)))
				if err != nil {
					continue
				}
				value, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(getContent(tag))))
				if err != nil {
					continue
				}
				for _, ref := range dec.stringRefs {
					if ref.Reference == string(key) {
						ref.Instance.Properties[ref.Property] = rbxfile.ValueSharedString(value)
					}
				}
			}
		}
	}

	for _, propRef := range dec.propRefs {
		dec.instLookup.Resolve(propRef)
	}

	return nil
}

func (dec *rdecoder) getItems(parent *rbxfile.Instance, tags []*documentTag) (instances []*rbxfile.Instance, properties map[string]rbxfile.Value) {
	properties = make(map[string]rbxfile.Value)
	hasProps := false

	for _, tag := range tags {
		switch tag.StartName {
		case "Item":
			className, ok := tag.AttrValue("class")
			if !ok {
				dec.document.Warnings = dec.document.Warnings.Append(errors.New("item with missing class attribute"))
				continue
			}

			instance := rbxfile.NewInstance(className)
			referent, ok := tag.AttrValue("referent")
			if ok && len(referent) > 0 {
				instance.Reference = referent
				if !rbxfile.IsEmptyReference(referent) {
					dec.instLookup[referent] = instance
				}
			}

			var children []*rbxfile.Instance
			children, instance.Properties = dec.getItems(instance, tag.Tags)
			instance.Children = make([]*rbxfile.Instance, len(children))
			for i, child := range children {
				instance.Children[i] = child
			}

			instances = append(instances, instance)

		case "Properties":
			if hasProps || parent == nil {
				continue
			}
			hasProps = true

			for _, property := range tag.Tags {
				name, value, ok := dec.getProperty(property, parent)
				if ok {
					properties[name] = value
				}
			}
		}
	}

	return instances, properties
}

// DecodeProperties decodes a list of tags as properties to a given instance.
// Returns a list of unresolved references.
func (c robloxCodec) DecodeProperties(tags []*documentTag, inst *rbxfile.Instance, refs rbxfile.References) (propRefs []rbxfile.PropRef) {
	dec := &rdecoder{
		codec:      c,
		instLookup: refs,
	}

	for _, property := range tags {
		name, value, ok := dec.getProperty(property, inst)
		if ok {
			inst.Properties[name] = value
		}
	}

	return dec.propRefs
}

func (dec *rdecoder) getProperty(tag *documentTag, instance *rbxfile.Instance) (name string, value rbxfile.Value, ok bool) {
	name, ok = tag.AttrValue("name")
	if !ok {
		return "", nil, false
	}

	// Guess property type from tag name.
	valueType, optional := dec.codec.GetCanonType(tag.StartName)
	if optional {
		tag, ok = dec.getOptional(tag, valueType)
		if !ok {
			return "", nil, false
		}
		if tag == nil {
			return name, rbxfile.None(valueType), true
		}
	}

	value, ok = dec.getValue(tag, valueType)
	if !ok {
		return "", nil, false
	}

	switch value := value.(type) {
	case rbxfile.ValueReference:
		if ref := getContent(tag); !rbxfile.IsEmptyReference(ref) {
			dec.propRefs = append(dec.propRefs, rbxfile.PropRef{
				Instance:  instance,
				Property:  name,
				Reference: ref,
			})
			return "", nil, false
		}
	case rbxfile.ValueSharedString:
		dec.stringRefs = append(dec.stringRefs, rbxfile.PropRef{
			Instance:  instance,
			Property:  name,
			Reference: string(value),
		})
		return "", nil, false
	}

	if optional {
		value = rbxfile.Some(value)
	}

	return name, value, ok
}

func (dec *rdecoder) getOptional(tag *documentTag, valueType rbxfile.Type) (subtag *documentTag, ok bool) {
	if len(tag.Tags) == 0 {
		return nil, true
	}
	switch valueType {
	case rbxfile.TypeCFrame:
		ok = components{
			"CFrame": &subtag, //TODO: case-sensitive?
		}.getFrom(tag)
	}
	return subtag, ok
}

// GetCanonTag converts a rbxfile.Type to its canonical tag name.
func (robloxCodec) GetCanonTag(valueType rbxfile.Type, optional bool) (canonTag string) {
	switch valueType {
	case rbxfile.TypeAxes:
		canonTag = "Axes"
	case rbxfile.TypeBinaryString:
		canonTag = "BinaryString"
	case rbxfile.TypeBool:
		canonTag = "bool"
	case rbxfile.TypeBrickColor:
		canonTag = "BrickColor"
	case rbxfile.TypeCFrame:
		canonTag = "CoordinateFrame"
	case rbxfile.TypeColor3:
		canonTag = "Color3"
	case rbxfile.TypeContent:
		canonTag = "Content"
	case rbxfile.TypeDouble:
		canonTag = "double"
	case rbxfile.TypeFaces:
		canonTag = "Faces"
	case rbxfile.TypeFloat:
		canonTag = "float"
	case rbxfile.TypeInt:
		canonTag = "int"
	case rbxfile.TypeProtectedString:
		canonTag = "ProtectedString"
	case rbxfile.TypeRay:
		canonTag = "Ray"
	case rbxfile.TypeReference:
		canonTag = "Object"
	case rbxfile.TypeString:
		canonTag = "string"
	case rbxfile.TypeToken:
		canonTag = "token"
	case rbxfile.TypeUDim:
		canonTag = "UDim"
	case rbxfile.TypeUDim2:
		canonTag = "UDim2"
	case rbxfile.TypeVector2:
		canonTag = "Vector2"
	case rbxfile.TypeVector2int16:
		canonTag = "Vector2int16"
	case rbxfile.TypeVector3:
		canonTag = "Vector3"
	case rbxfile.TypeVector3int16:
		canonTag = "Vector3int16"
	case rbxfile.TypeNumberSequence:
		canonTag = "NumberSequence"
	case rbxfile.TypeColorSequence:
		canonTag = "ColorSequence"
	case rbxfile.TypeNumberRange:
		canonTag = "NumberRange"
	case rbxfile.TypeRect:
		canonTag = "Rect2D"
	case rbxfile.TypePhysicalProperties:
		canonTag = "PhysicalProperties"
	case rbxfile.TypeColor3uint8:
		canonTag = "Color3uint8"
	case rbxfile.TypeInt64:
		canonTag = "int64"
	case rbxfile.TypeSharedString:
		canonTag = "SharedString"
	case rbxfile.TypeUniqueId:
		canonTag = "UniqueId"
	}
	if optional {
		canonTag = "Optional" + canonTag
	}
	return canonTag
}

// GetCanonType converts a string from a tag name to a rbxfile.Type.
func (robloxCodec) GetCanonType(valueType string) (canonType rbxfile.Type, optional bool) {
	valueType = strings.ToLower(valueType)
	if strings.HasPrefix(valueType, "optional") {
		valueType = strings.TrimPrefix(valueType, "optional")
		optional = true
	}
	switch valueType {
	case "axes":
		canonType = rbxfile.TypeAxes
	case "binarystring":
		canonType = rbxfile.TypeBinaryString
	case "bool":
		canonType = rbxfile.TypeBool
	case "brickcolor":
		canonType = rbxfile.TypeBrickColor
	case "cframe", "coordinateframe":
		canonType = rbxfile.TypeCFrame
	case "color3":
		canonType = rbxfile.TypeColor3
	case "content":
		canonType = rbxfile.TypeContent
	case "double":
		canonType = rbxfile.TypeDouble
	case "faces":
		canonType = rbxfile.TypeFaces
	case "float":
		canonType = rbxfile.TypeFloat
	case "int":
		canonType = rbxfile.TypeInt
	case "protectedstring":
		canonType = rbxfile.TypeProtectedString
	case "ray":
		canonType = rbxfile.TypeRay
	case "object", "ref":
		canonType = rbxfile.TypeReference
	case "string":
		canonType = rbxfile.TypeString
	case "token":
		canonType = rbxfile.TypeToken
	case "udim":
		canonType = rbxfile.TypeUDim
	case "udim2":
		canonType = rbxfile.TypeUDim2
	case "vector2":
		canonType = rbxfile.TypeVector2
	case "vector2int16":
		canonType = rbxfile.TypeVector2int16
	case "vector3":
		canonType = rbxfile.TypeVector3
	case "vector3int16":
		canonType = rbxfile.TypeVector3int16
	case "numbersequence":
		canonType = rbxfile.TypeNumberSequence
	case "colorsequence":
		canonType = rbxfile.TypeColorSequence
	case "numberrange":
		canonType = rbxfile.TypeNumberRange
	case "rect", "rect2d":
		canonType = rbxfile.TypeRect
	case "physicalproperties":
		canonType = rbxfile.TypePhysicalProperties
	case "color3uint8":
		canonType = rbxfile.TypeColor3uint8
	case "int64":
		canonType = rbxfile.TypeInt64
	case "sharedstring":
		canonType = rbxfile.TypeSharedString
	case "uniqueid":
		canonType = rbxfile.TypeUniqueId
	}
	return canonType, optional
}

// Gets a rbxfile.Value from a property tag, using valueType to determine how
// the tag is interpreted. valueType must be an existing type as it appears in
// the API dump. If guessing the type, it should be converted to one of these
// first.
func (dec *rdecoder) getValue(tag *documentTag, valueType rbxfile.Type) (value rbxfile.Value, ok bool) {
	switch valueType {
	case rbxfile.TypeAxes:
		var bits int32
		ok := components{
			"axes": &bits,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueAxes{}, true
		}
		return rbxfile.ValueAxes{
			X: bits&(1<<0) > 0,
			Y: bits&(1<<1) > 0,
			Z: bits&(1<<2) > 0,
		}, true

	case rbxfile.TypeBinaryString:
		d := base64.NewDecoder(base64.StdEncoding, strings.NewReader(getContent(tag)))
		v, err := io.ReadAll(d)
		if err != nil {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueBinaryString(nil), true
		}
		return rbxfile.ValueBinaryString(v), true

	case rbxfile.TypeBool:
		switch getContent(tag) {
		case "false", "False", "FALSE":
			return rbxfile.ValueBool(false), true
		case "true", "True", "TRUE":
			return rbxfile.ValueBool(true), true
		}
		if dec.codec.DiscardInvalidProperties {
			return nil, false
		}
		return rbxfile.ValueBool(false), true

	case rbxfile.TypeBrickColor:
		v, err := strconv.ParseUint(getContent(tag), 10, 32)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueBrickColor(0), true
		}
		return rbxfile.ValueBrickColor(v), true

	case rbxfile.TypeCFrame:
		var v rbxfile.ValueCFrame
		ok := components{
			"X":   &v.Position.X,
			"Y":   &v.Position.Y,
			"Z":   &v.Position.Z,
			"R00": &v.Rotation[0],
			"R01": &v.Rotation[1],
			"R02": &v.Rotation[2],
			"R10": &v.Rotation[3],
			"R11": &v.Rotation[4],
			"R12": &v.Rotation[5],
			"R20": &v.Rotation[6],
			"R21": &v.Rotation[7],
			"R22": &v.Rotation[8],
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueCFrame{}, true
		}
		return v, true

	case rbxfile.TypeColor3:
		content := getContent(tag)
		if len(content) > 0 {
			v, err := strconv.ParseUint(content, 10, 32)
			if err != nil && !errors.Is(err, strconv.ErrRange) {
				if dec.codec.DiscardInvalidProperties {
					return nil, false
				}
				return rbxfile.ValueColor3{}, true
			}
			return rbxfile.ValueColor3{
				R: float32(v&0x00FF0000>>16) / 255,
				G: float32(v&0x0000FF00>>8) / 255,
				B: float32(v&0x000000FF) / 255,
			}, true
		} else {
			//DIFF: If any tags are missing, entire value defaults.
			var v rbxfile.ValueColor3
			ok := components{
				"R": &v.R,
				"G": &v.G,
				"B": &v.B,
			}.getFrom(tag)
			if !ok {
				if dec.codec.DiscardInvalidProperties {
					return nil, false
				}
				return rbxfile.ValueColor3{}, true
			}
			return v, true
		}

	case rbxfile.TypeContent:
		if tag.CData == nil && len(tag.Text) > 0 || tag.CData != nil && len(tag.CData) > 0 {
			// Succeeds if CData is not nil but empty, even if Text is not
			// empty. This is correct according to Roblox's codec.
			return nil, false
		}
	loop:
		for _, subtag := range tag.Tags {
			switch subtag.StartName {
			case "binary":
				dec.document.Warnings = dec.document.Warnings.Append(errors.New("not reading binary data"))
				return rbxfile.ValueContent(nil), true
			case "hash":
				// Ignored.
				return rbxfile.ValueContent(nil), true
			case "null":
				//DIFF: If null tag has content, then `tag expected` error is
				//thrown.
				return rbxfile.ValueContent(nil), true
			case "url":
				return rbxfile.ValueContent(getContent(subtag)), true
			default:
				//DIFF: Throws error `TextXmlParser::parse - Unknown tag ''.`
				break loop
			}
		}
		//DIFF: When tag has no subtags, attempts to read end tag as a subtag,
		//erroneously throwing an "unknown tag" error.
		if dec.codec.DiscardInvalidProperties {
			return nil, false
		}
		return rbxfile.ValueContent(nil), true

	case rbxfile.TypeDouble:
		// TODO: check inf, nan, and overflow. ParseFloat reads special numbers
		// in several forms. Depending on how Roblox parses such values, we may
		// have to catch these forms early and treat them as invalid.
		v, err := strconv.ParseFloat(getContent(tag), 64)
		if err != nil {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueDouble(0), true
		}
		return rbxfile.ValueDouble(v), true

	case rbxfile.TypeFaces:
		var bits int32
		ok := components{
			"faces": &bits,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueFaces{}, true
		}
		return rbxfile.ValueFaces{
			Right:  bits&(1<<0) > 0,
			Top:    bits&(1<<1) > 0,
			Back:   bits&(1<<2) > 0,
			Left:   bits&(1<<3) > 0,
			Bottom: bits&(1<<4) > 0,
			Front:  bits&(1<<5) > 0,
		}, true

	case rbxfile.TypeFloat:
		v, err := strconv.ParseFloat(getContent(tag), 32)
		if err != nil {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueFloat(0), true
		}
		return rbxfile.ValueFloat(v), true

	case rbxfile.TypeInt:
		v, err := strconv.ParseInt(getContent(tag), 10, 32)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueInt(0), true
		}
		return rbxfile.ValueInt(v), true

	case rbxfile.TypeProtectedString:
		return rbxfile.ValueProtectedString(getContent(tag)), true

	case rbxfile.TypeRay:
		var origin, direction *documentTag
		ok := components{
			"origin":    &origin,
			"direction": &direction,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueRay{}, true
		}
		var v rbxfile.ValueRay
		ok = components{
			"X": &v.Origin.X,
			"Y": &v.Origin.Y,
			"Z": &v.Origin.Z,
		}.getFrom(origin)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueRay{}, true
		}
		ok = components{
			"X": &v.Direction.X,
			"Y": &v.Direction.Y,
			"Z": &v.Direction.Z,
		}.getFrom(direction)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueRay{}, true
		}
		return v, true

	case rbxfile.TypeReference:
		// Return empty ValueReference; this signals that the value will be
		// acquired later.
		return rbxfile.ValueReference{}, true

	case rbxfile.TypeString:
		return rbxfile.ValueString(getContent(tag)), true

	case rbxfile.TypeToken:
		v, err := strconv.ParseInt(getContent(tag), 10, 32)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueToken(0), true
		}
		return rbxfile.ValueToken(v), true

	case rbxfile.TypeUDim:
		var v rbxfile.ValueUDim
		ok := components{
			"S": &v.Scale,
			"O": &v.Offset,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueUDim{}, true
		}
		return v, true

	case rbxfile.TypeUDim2:
		// DIFF: UDim2 is initialized with odd values
		var v rbxfile.ValueUDim2
		ok := components{
			"XS": &v.X.Scale,
			"XO": &v.X.Offset,
			"YS": &v.Y.Scale,
			"YO": &v.Y.Offset,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueUDim2{}, true
		}
		return v, true

	case rbxfile.TypeVector2:
		var v rbxfile.ValueVector2
		ok := components{
			"X": &v.X,
			"Y": &v.Y,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueVector2{}, true
		}
		return v, true

	case rbxfile.TypeVector2int16:
		// Unknown; guessed
		var v rbxfile.ValueVector2int16
		ok := components{
			"X": &v.X,
			"Y": &v.Y,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueVector2int16{}, true
		}
		return v, true

	case rbxfile.TypeVector3:
		var v rbxfile.ValueVector3
		ok := components{
			"X": &v.X,
			"Y": &v.Y,
			"Z": &v.Z,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueVector3{}, true
		}
		return v, true

	case rbxfile.TypeVector3int16:
		// Unknown; guessed
		var v rbxfile.ValueVector3int16
		ok := components{
			"X": &v.X,
			"Y": &v.Y,
			"Z": &v.Z,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueVector3int16{}, true
		}
		return v, true

	case rbxfile.TypeNumberSequence:
		b := []byte(getContent(tag))
		v := make(rbxfile.ValueNumberSequence, 0, 4)
		for i := 0; i < len(b); {
			nsk := rbxfile.ValueNumberSequenceKeypoint{}
			nsk.Time, i = scanFloat(b, i)
			nsk.Value, i = scanFloat(b, i)
			nsk.Envelope, i = scanFloat(b, i)
			if i < 0 {
				if dec.codec.DiscardInvalidProperties {
					return nil, false
				}
				return rbxfile.ValueNumberSequence(nil), true
			}
			v = append(v, nsk)
		}
		return v, true

	case rbxfile.TypeColorSequence:
		b := []byte(getContent(tag))
		v := make(rbxfile.ValueColorSequence, 0, 4)
		for i := 0; i < len(b); {
			csk := rbxfile.ValueColorSequenceKeypoint{}
			csk.Time, i = scanFloat(b, i)
			csk.Value.R, i = scanFloat(b, i)
			csk.Value.G, i = scanFloat(b, i)
			csk.Value.B, i = scanFloat(b, i)
			csk.Envelope, i = scanFloat(b, i)
			if i < 0 {
				if dec.codec.DiscardInvalidProperties {
					return nil, false
				}
				return rbxfile.ValueColorSequence(nil), true
			}
			v = append(v, csk)
		}
		return v, true

	case rbxfile.TypeNumberRange:
		b := []byte(getContent(tag))
		var v rbxfile.ValueNumberRange
		i := 0
		v.Min, i = scanFloat(b, i)
		v.Max, i = scanFloat(b, i)
		if i < 0 {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueNumberRange{}, true
		}
		return v, true

	case rbxfile.TypeRect:
		var min, max *documentTag
		ok := components{
			"min": &min,
			"max": &max,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueRect{}, true
		}
		var v rbxfile.ValueRect
		ok = components{
			"X": &v.Min.X,
			"Y": &v.Min.Y,
		}.getFrom(min)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueRect{}, true
		}
		ok = components{
			"X": &v.Max.X,
			"Y": &v.Max.Y,
		}.getFrom(max)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueRect{}, true
		}
		return v, true

	case rbxfile.TypePhysicalProperties:
		var v rbxfile.ValuePhysicalProperties
		var cp *documentTag
		ok := components{
			"CustomPhysics": &cp,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValuePhysicalProperties{}, true
		}
		vb, ok := dec.getValue(cp, rbxfile.TypeBool)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValuePhysicalProperties{}, true
		}
		v.CustomPhysics = bool(vb.(rbxfile.ValueBool))
		if !v.CustomPhysics {
			return v, true
		}
		ok = components{
			"Density":          &v.Density,
			"Friction":         &v.Friction,
			"Elasticity":       &v.Elasticity,
			"FrictionWeight":   &v.FrictionWeight,
			"ElasticityWeight": &v.ElasticityWeight,
		}.getFrom(tag)
		if !ok {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return v, true
		}
		return v, true

	case rbxfile.TypeColor3uint8:
		content := getContent(tag)
		if len(content) > 0 {
			v, err := strconv.ParseUint(content, 10, 32)
			if err != nil && !errors.Is(err, strconv.ErrRange) {
				if dec.codec.DiscardInvalidProperties {
					return nil, false
				}
				return rbxfile.ValueColor3uint8{}, true
			}
			return rbxfile.ValueColor3uint8{
				R: byte(v & 0x00FF0000 >> 16),
				G: byte(v & 0x0000FF00 >> 8),
				B: byte(v & 0x000000FF),
			}, true
		} else {
			//DIFF: If any tags are missing, entire value defaults.
			var v rbxfile.ValueColor3uint8
			ok := components{
				"R": &v.R,
				"G": &v.G,
				"B": &v.B,
			}.getFrom(tag)
			if !ok {
				if dec.codec.DiscardInvalidProperties {
					return nil, false
				}
				return rbxfile.ValueColor3uint8{}, true
			}
			return v, true
		}

	case rbxfile.TypeInt64:
		v, err := strconv.ParseInt(getContent(tag), 10, 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueInt64(0), true
		}
		return rbxfile.ValueInt64(v), true

	case rbxfile.TypeSharedString:
		v, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(getContent(tag))))
		if err != nil {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueSharedString(nil), true
		}
		return rbxfile.ValueSharedString(v), true

	case rbxfile.TypeUniqueId:
		//TODO: Not included in model format.
		var v rbxfile.ValueUniqueId
		text := tag.Text
		if len(text) > 32 {
			text = text[:32]
		}
		b, err := hex.DecodeString(text)
		if err != nil || len(b) < 16 {
			if dec.codec.DiscardInvalidProperties {
				return nil, false
			}
			return rbxfile.ValueUniqueId{}, true
		}
		v.Random = int64(binary.BigEndian.Uint64(b[0:8]))
		v.Time = binary.BigEndian.Uint32(b[8:12])
		v.Index = binary.BigEndian.Uint32(b[12:16])
		return v, true
	}
	return nil, false
}

func scanFloat(b []byte, i int) (float32, int) {
	if i < 0 || i >= len(b) {
		return 0, -1
	}
	s := i
	for ; i < len(b); i++ {
		if isSpace(b[i]) {
			f, err := strconv.ParseFloat(string(b[s:i]), 32)
			if err != nil {
				return 0, -1
			}
			for ; i < len(b); i++ {
				if !isSpace(b[i]) {
					break
				}
			}
			return float32(f), i
		}
	}
	return 0, -1
}

type components map[string]interface{}

func (c components) getFrom(tag *documentTag) (ok bool) {
	if tag == nil {
		return false
	}

	// Used to ensure that only the first matched tag is selected.
	d := make(map[string]bool, 12)

	for _, subtag := range tag.Tags {
		if p, ok := c[subtag.StartName]; ok && !d[subtag.StartName] {
			d[subtag.StartName] = true
			switch v := p.(type) {
			case *uint8:
				// Parsed as int32 % 256.
				n, err := strconv.ParseInt(getContent(subtag), 10, 32)
				*v = uint8(n % 256)
				if err != nil {
					if errors.Is(err, strconv.ErrRange) {
						return true
					}
					return false
				}
			case *int16:
				n, err := strconv.ParseInt(getContent(subtag), 10, 16)
				*v = int16(n)
				if err != nil {
					if errors.Is(err, strconv.ErrRange) {
						return true
					}
					return false
				}
			case *int32:
				n, err := strconv.ParseInt(getContent(subtag), 10, 32)
				*v = int32(n)
				if err != nil {
					if errors.Is(err, strconv.ErrRange) {
						return true
					}
					return false
				}
			case *float32:
				if n, err := strconv.ParseFloat(getContent(subtag), 32); err == nil {
					*v = float32(n)
				}
			case **documentTag:
				*v = subtag
			}
		}
	}
	// Fail if not all components have been found.
	return len(d) == len(c)
}

// Reads either the CData or the text of a tag.
func getContent(tag *documentTag) string {
	if tag.CData != nil {
		// CData is preferred even if it is empty
		return string(tag.CData)
	}
	return tag.Text
}

type rencoder struct {
	root          *rbxfile.Root
	codec         robloxCodec
	document      *documentRoot
	refs          rbxfile.References
	sharedStrings map[string][]byte
	err           error
}

func (c robloxCodec) Encode(root *rbxfile.Root) (document *documentRoot, err error) {
	enc := &rencoder{
		root:          root,
		codec:         c,
		refs:          make(rbxfile.References),
		sharedStrings: map[string][]byte{},
	}

	enc.encode()
	return enc.document, enc.err

}

type sortTagsByNameAttr []*documentTag

func (t sortTagsByNameAttr) Len() int {
	return len(t)
}
func (t sortTagsByNameAttr) Less(i, j int) bool {
	return t[i].Attr[0].Value < t[j].Attr[0].Value
}
func (t sortTagsByNameAttr) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

type wrapWriter struct {
	l int
	c int
	w io.Writer
}

func newWrapWriter(length int, w io.Writer) *wrapWriter {
	return &wrapWriter{l: length, w: w}
}

func (w *wrapWriter) Write(p []byte) (n int, err error) {
	for w.c+len(p) >= w.l {
		i := w.l - w.c
		nn, err := w.w.Write(p[:i])
		n += nn
		if err != nil {
			return n, err
		}
		nn, err = w.w.Write([]byte{'\n'})
		if err != nil {
			return n, err
		}
		w.c = 0
		p = p[i:]
	}
	if len(p) > 0 {
		nn, err := w.w.Write(p)
		n += nn
		w.c += nn
		return n, err
	}
	return n, nil
}

func (enc *rencoder) encode() {
	enc.document = &documentRoot{
		Prefix: "",
		Indent: "\t",
		Suffix: "",
		Root:   newRoot(),
	}
	if !enc.codec.ExcludeMetadata {
		enc.document.Root.Tags = make([]*documentTag, 0, len(enc.root.Metadata))
		for key, value := range enc.root.Metadata {
			enc.document.Root.Tags = append(enc.document.Root.Tags, &documentTag{
				StartName: "Meta",
				Attr:      []documentAttr{{Name: "name", Value: key}},
				Text:      value,
			})
		}
		sort.Sort(sortTagsByNameAttr(enc.document.Root.Tags))
	}
	if !enc.codec.ExcludeExternal {
		enc.document.Root.Tags = append(enc.document.Root.Tags,
			&documentTag{StartName: "External", Text: "null"},
			&documentTag{StartName: "External", Text: "nil"},
		)
	}

	for _, instance := range enc.root.Instances {
		enc.encodeInstance(instance, enc.document.Root)
	}

	if len(enc.sharedStrings) > 0 {
		//TODO: Tags are sorted by hash. Check if they're sorted pre- or
		//post-base64 encoding.
		keys := make([]string, 0, len(enc.sharedStrings))
		for key := range enc.sharedStrings {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		tag := &documentTag{StartName: "SharedStrings", Tags: make([]*documentTag, len(keys))}
		var s strings.Builder
		for i, key := range keys {
			b64 := base64.NewEncoder(base64.StdEncoding, newWrapWriter(72, &s))
			b64.Write(enc.sharedStrings[key])
			b64.Close()
			tag.Tags[i] = &documentTag{
				StartName: "SharedString",
				Attr: []documentAttr{{
					Name:  "md5",
					Value: base64.StdEncoding.EncodeToString([]byte(key)),
				}},
				Text: s.String(),
			}
			s.Reset()
		}
		enc.document.Root.Tags = append(enc.document.Root.Tags, tag)
	}
}

func (enc *rencoder) encodeInstance(instance *rbxfile.Instance, parent *documentTag) {
	ref := enc.refs.Get(instance)
	properties := enc.encodeProperties(instance)
	item := newItem(instance.ClassName, ref, properties...)
	if enc.codec.ExcludeReferent {
		item.SetAttrValue("referent", "")
	}
	parent.Tags = append(parent.Tags, item)

	for _, child := range instance.Children {
		enc.encodeInstance(child, item)
	}
}

func (c robloxCodec) EncodeProperties(instance *rbxfile.Instance) (properties []*documentTag) {
	enc := &rencoder{codec: c}
	return enc.encodeProperties(instance)
}

func (enc *rencoder) encodeProperties(instance *rbxfile.Instance) (properties []*documentTag) {
	// Sort properties by name
	sorted := make([]string, 0, len(instance.Properties))
	for name := range instance.Properties {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	for _, name := range sorted {
		value := instance.Properties[name]
		tag := enc.encodeProperty(value)
		if tag != nil {
			tag.Attr = []documentAttr{{Name: "name", Value: name}}
			properties = append(properties, tag)
		}
	}

	return properties
}

func (enc *rencoder) encodeProperty(value rbxfile.Value) *documentTag {
	switch value := value.(type) {
	case rbxfile.ValueAxes:
		var n uint64
		for i, b := range []bool{value.X, value.Y, value.Z} {
			if b {
				n |= (1 << uint(i))
			}
		}
		return &documentTag{
			StartName: "Axes",
			Tags: []*documentTag{
				{
					StartName: "axes",
					NoIndent:  true,
					Text:      strconv.FormatUint(n, 10),
				},
			},
		}

	case rbxfile.ValueBinaryString:
		buf := new(bytes.Buffer)
		sw := &lineSplit{w: buf, s: 72, n: 72}
		bw := base64.NewEncoder(base64.StdEncoding, sw)
		bw.Write([]byte(value))
		bw.Close()
		tag := &documentTag{
			StartName: "BinaryString",
			NoIndent:  true,
		}
		encodeContent(tag, buf.String())
		return tag

	case rbxfile.ValueBool:
		var v string
		if value {
			v = "true"
		} else {
			v = "false"
		}
		return &documentTag{
			StartName: "bool",
			NoIndent:  true,
			Text:      v,
		}

	case rbxfile.ValueBrickColor:
		return &documentTag{
			StartName: "int",
			NoIndent:  true,
			Text:      strconv.FormatUint(uint64(value), 10),
		}

	case rbxfile.ValueCFrame:
		return &documentTag{
			StartName: "CoordinateFrame",
			Tags: []*documentTag{
				{StartName: "X", NoIndent: true, Text: encodeFloat(value.Position.X)},
				{StartName: "Y", NoIndent: true, Text: encodeFloat(value.Position.Y)},
				{StartName: "Z", NoIndent: true, Text: encodeFloat(value.Position.Z)},
				{StartName: "R00", NoIndent: true, Text: encodeFloat(value.Rotation[0])},
				{StartName: "R01", NoIndent: true, Text: encodeFloat(value.Rotation[1])},
				{StartName: "R02", NoIndent: true, Text: encodeFloat(value.Rotation[2])},
				{StartName: "R10", NoIndent: true, Text: encodeFloat(value.Rotation[3])},
				{StartName: "R11", NoIndent: true, Text: encodeFloat(value.Rotation[4])},
				{StartName: "R12", NoIndent: true, Text: encodeFloat(value.Rotation[5])},
				{StartName: "R20", NoIndent: true, Text: encodeFloat(value.Rotation[6])},
				{StartName: "R21", NoIndent: true, Text: encodeFloat(value.Rotation[7])},
				{StartName: "R22", NoIndent: true, Text: encodeFloat(value.Rotation[8])},
			},
		}

	case rbxfile.ValueColor3:
		return &documentTag{
			StartName: "Color3",
			Tags: []*documentTag{
				{StartName: "R", NoIndent: true, Text: encodeFloat(value.R)},
				{StartName: "G", NoIndent: true, Text: encodeFloat(value.G)},
				{StartName: "B", NoIndent: true, Text: encodeFloat(value.B)},
			},
		}

	case rbxfile.ValueContent:
		tag := &documentTag{
			StartName: "Content",
			NoIndent:  true,
			Tags: []*documentTag{
				{
					StartName: "",
					NoIndent:  true,
				},
			},
		}
		if len(value) == 0 {
			tag.Tags[0].StartName = "null"
		} else {
			tag.Tags[0].StartName = "url"
			tag.Tags[0].Text = string(value)
		}
		return tag

	case rbxfile.ValueDouble:
		return &documentTag{
			StartName: "double",
			NoIndent:  true,
			Text:      encodeDouble(float64(value)),
		}

	case rbxfile.ValueFaces:
		var n uint64
		for i, b := range []bool{value.Right, value.Top, value.Back, value.Left, value.Bottom, value.Front} {
			if b {
				n |= (1 << uint(i))
			}
		}
		return &documentTag{
			StartName: "Faces",
			Tags: []*documentTag{
				{
					StartName: "faces",
					NoIndent:  true,
					Text:      strconv.FormatUint(n, 10),
				},
			},
		}

	case rbxfile.ValueFloat:
		return &documentTag{
			StartName: "float",
			NoIndent:  true,
			Text:      encodeFloat(float32(value)),
		}

	case rbxfile.ValueInt:
		return &documentTag{
			StartName: "int",
			NoIndent:  true,
			Text:      strconv.FormatInt(int64(value), 10),
		}

	case rbxfile.ValueProtectedString:
		tag := &documentTag{
			StartName: "ProtectedString",
			NoIndent:  true,
		}
		encodeContent(tag, string(value))
		return tag

	case rbxfile.ValueRay:
		return &documentTag{
			StartName: "Ray",
			Tags: []*documentTag{
				{
					StartName: "origin",
					Tags: []*documentTag{
						{StartName: "X", NoIndent: true, Text: encodeFloat(value.Origin.X)},
						{StartName: "Y", NoIndent: true, Text: encodeFloat(value.Origin.Y)},
						{StartName: "Z", NoIndent: true, Text: encodeFloat(value.Origin.Z)},
					},
				},
				{
					StartName: "direction",
					Tags: []*documentTag{
						{StartName: "X", NoIndent: true, Text: encodeFloat(value.Origin.X)},
						{StartName: "Y", NoIndent: true, Text: encodeFloat(value.Origin.Y)},
						{StartName: "Z", NoIndent: true, Text: encodeFloat(value.Origin.Z)},
					},
				},
			},
		}

	case rbxfile.ValueReference:
		tag := &documentTag{
			StartName: "Ref",
			NoIndent:  true,
		}

		referent := value.Instance
		if referent != nil {
			tag.Text = enc.refs.Get(referent)
		} else {
			tag.Text = "null"
		}
		return tag

	case rbxfile.ValueString:
		return &documentTag{
			StartName: "string",
			NoIndent:  true,
			Text:      string(value),
		}

	case rbxfile.ValueToken:
		return &documentTag{
			StartName: "token",
			NoIndent:  true,
			Text:      strconv.FormatUint(uint64(value), 10),
		}

	case rbxfile.ValueUDim:
		return &documentTag{
			StartName: "UDim",
			Tags: []*documentTag{
				{StartName: "S", NoIndent: true, Text: encodeFloat(value.Scale)},
				{StartName: "O", NoIndent: true, Text: strconv.FormatInt(int64(value.Offset), 10)},
			},
		}

	case rbxfile.ValueUDim2:
		return &documentTag{
			StartName: "UDim2",
			Tags: []*documentTag{
				{StartName: "XS", NoIndent: true, Text: encodeFloat(value.X.Scale)},
				{StartName: "XO", NoIndent: true, Text: strconv.FormatInt(int64(value.X.Offset), 10)},
				{StartName: "YS", NoIndent: true, Text: encodeFloat(value.Y.Scale)},
				{StartName: "YO", NoIndent: true, Text: strconv.FormatInt(int64(value.Y.Offset), 10)},
			},
		}

	case rbxfile.ValueVector2:
		return &documentTag{
			StartName: "Vector2",
			Tags: []*documentTag{
				{StartName: "X", NoIndent: true, Text: encodeFloat(value.X)},
				{StartName: "Y", NoIndent: true, Text: encodeFloat(value.Y)},
			},
		}

	case rbxfile.ValueVector2int16:
		return &documentTag{
			StartName: "Vector2int16",
			Tags: []*documentTag{
				{StartName: "X", NoIndent: true, Text: strconv.FormatInt(int64(value.X), 10)},
				{StartName: "Y", NoIndent: true, Text: strconv.FormatInt(int64(value.Y), 10)},
			},
		}

	case rbxfile.ValueVector3:
		return &documentTag{
			StartName: "Vector3",
			Tags: []*documentTag{
				{StartName: "X", NoIndent: true, Text: encodeFloat(value.X)},
				{StartName: "Y", NoIndent: true, Text: encodeFloat(value.Y)},
				{StartName: "Z", NoIndent: true, Text: encodeFloat(value.Z)},
			},
		}

	case rbxfile.ValueVector3int16:
		return &documentTag{
			StartName: "Vector3int16",
			Tags: []*documentTag{
				{StartName: "X", NoIndent: true, Text: strconv.FormatInt(int64(value.X), 10)},
				{StartName: "Y", NoIndent: true, Text: strconv.FormatInt(int64(value.Y), 10)},
				{StartName: "Z", NoIndent: true, Text: strconv.FormatInt(int64(value.Z), 10)},
			},
		}

	case rbxfile.ValueNumberSequence:
		b := make([]byte, 0, 16)
		for _, nsk := range value {
			b = append(b, []byte(encodeFloatPrec(nsk.Time, 6))...)
			b = append(b, ' ')
			b = append(b, []byte(encodeFloatPrec(nsk.Value, 6))...)
			b = append(b, ' ')
			b = append(b, []byte(encodeFloatPrec(nsk.Envelope, 6))...)
			b = append(b, ' ')
		}
		return &documentTag{
			StartName: "NumberSequence",
			Text:      string(b),
		}

	case rbxfile.ValueColorSequence:
		b := make([]byte, 0, 32)
		for _, csk := range value {
			b = append(b, []byte(encodeFloatPrec(csk.Time, 6))...)
			b = append(b, ' ')
			b = append(b, []byte(encodeFloatPrec(csk.Value.R, 6))...)
			b = append(b, ' ')
			b = append(b, []byte(encodeFloatPrec(csk.Value.G, 6))...)
			b = append(b, ' ')
			b = append(b, []byte(encodeFloatPrec(csk.Value.B, 6))...)
			b = append(b, ' ')
			b = append(b, []byte(encodeFloatPrec(csk.Envelope, 6))...)
			b = append(b, ' ')
		}
		return &documentTag{
			StartName: "ColorSequence",
			Text:      string(b),
		}

	case rbxfile.ValueNumberRange:
		b := make([]byte, 0, 8)
		b = append(b, []byte(encodeFloatPrec(value.Min, 6))...)
		b = append(b, ' ')
		b = append(b, []byte(encodeFloatPrec(value.Max, 6))...)
		b = append(b, ' ')
		return &documentTag{
			StartName: "NumberRange",
			Text:      string(b),
		}

	case rbxfile.ValueRect:
		return &documentTag{
			StartName: "Rect2D",
			Tags: []*documentTag{
				{
					StartName: "min",
					Tags: []*documentTag{
						{StartName: "X", NoIndent: true, Text: encodeFloat(value.Min.X)},
						{StartName: "Y", NoIndent: true, Text: encodeFloat(value.Min.Y)},
					},
				},
				{
					StartName: "max",
					Tags: []*documentTag{
						{StartName: "X", NoIndent: true, Text: encodeFloat(value.Max.X)},
						{StartName: "Y", NoIndent: true, Text: encodeFloat(value.Max.Y)},
					},
				},
			},
		}

	case rbxfile.ValuePhysicalProperties:
		if value.CustomPhysics {
			return &documentTag{
				StartName: "PhysicalProperties",
				Tags: []*documentTag{
					{StartName: "CustomPhysics", Text: "true"},
					{StartName: "Density", Text: encodeFloat(value.Density)},
					{StartName: "Friction", Text: encodeFloat(value.Friction)},
					{StartName: "Elasticity", Text: encodeFloat(value.Elasticity)},
					{StartName: "FrictionWeight", Text: encodeFloat(value.FrictionWeight)},
					{StartName: "ElasticityWeight", Text: encodeFloat(value.ElasticityWeight)},
				},
			}
		} else {
			return &documentTag{
				StartName: "PhysicalProperties",
				Tags: []*documentTag{
					{StartName: "CustomPhysics", Text: "false"},
				},
			}
		}

	case rbxfile.ValueColor3uint8:
		r := uint64(value.R)
		g := uint64(value.G)
		b := uint64(value.B)
		return &documentTag{
			StartName: "Color3uint8",
			NoIndent:  true,
			Text:      strconv.FormatUint(0xFF<<24|r<<16|g<<8|b, 10),
		}

	case rbxfile.ValueInt64:
		return &documentTag{
			StartName: "int64",
			NoIndent:  true,
			Text:      strconv.FormatInt(int64(value), 10),
		}

	case rbxfile.ValueSharedString:
		sum := blake2b.Sum256(value)
		hash := string(sum[:16])
		if _, ok := enc.sharedStrings[hash]; !ok {
			enc.sharedStrings[hash] = []byte(value)
		}

		buf := new(bytes.Buffer)
		sw := &lineSplit{w: buf, s: 72, n: 72}
		bw := base64.NewEncoder(base64.StdEncoding, sw)
		bw.Write(sum[:16])
		bw.Close()
		tag := &documentTag{
			StartName: "SharedString",
			NoIndent:  true,
		}
		encodeContent(tag, buf.String())
		return tag

	case rbxfile.ValueOptional:
		parent := &documentTag{
			StartName: enc.codec.GetCanonTag(value.ValueType(), true),
		}
		switch value := value.Value().(type) {
		case nil:
			parent.NoIndent = true

		case rbxfile.ValueCFrame:
			tag := enc.encodeProperty(value)
			tag.StartName = "CFrame"
			parent.Tags = append(parent.Tags, tag)
		}
		return parent

	case rbxfile.ValueUniqueId:
		var b [16]byte
		binary.BigEndian.PutUint64(b[0:8], uint64(value.Random))
		binary.BigEndian.PutUint32(b[8:12], value.Time)
		binary.BigEndian.PutUint32(b[12:16], value.Index)
		return &documentTag{
			StartName: "UniqueId",
			Text:      hex.EncodeToString(b[:]),
		}
	}

	return nil
}

type lineSplit struct {
	w io.Writer
	s int
	n int
}

func (l *lineSplit) Write(p []byte) (n int, err error) {
	for i := 0; ; {
		var q []byte
		if len(p[i:]) < l.n {
			q = p[i:]
		} else {
			q = p[i : i+l.n]
		}
		n, err = l.w.Write(q)
		if n < len(q) {
			return
		}
		l.n -= len(q)
		i += len(q)
		if i >= len(p) {
			break
		}
		if l.n <= 0 {
			_, e := l.w.Write([]byte{'\n'})
			if e != nil {
				return
			}
			l.n = l.s
		}
	}
	return
}

func encodeFloat(f float32) string {
	return fixFloatExp(strconv.FormatFloat(float64(f), 'g', 9, 32), 3)
}

func encodeFloatPrec(f float32, prec int) string {
	return fixFloatExp(strconv.FormatFloat(float64(f), 'g', prec, 32), 3)
}

func fixFloatExp(s string, n int) string {
	if e := strings.Index(s, "e"); e >= 0 {
		// Adjust exponent to have length of at least n, using leading zeros.
		exp := s[e+2:]
		if len(exp) < n {
			s = s[:e+2] + strings.Repeat("0", n-len(exp)) + exp
		}
	}
	return s
}

func encodeDouble(f float64) string {
	return strconv.FormatFloat(f, 'g', 9, 64)
}

func encodeContent(tag *documentTag, text string) {
	if strings.Index(text, "]]>") >= 0 {
		tag.CData = []byte(text)
		return
	}
	tag.Text = text
}
