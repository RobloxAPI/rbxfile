// The json package is used to encode and decode rbxfile objects to the JSON
// format.
package json

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/robloxapi/rbxfile"
	"io/ioutil"
)

func Encode(root *rbxfile.Root) (b []byte, err error) {
	return json.Marshal(RootToJSONInterface(root))
}

func Decode(b []byte) (root *rbxfile.Root, err error) {
	var v interface{}
	err = json.Unmarshal(b, &v)
	if err != nil {
		return nil, err
	}
	root, ok := RootFromJSONInterface(v)
	if !ok {
		return nil, errors.New("invalid JSON Root object")
	}
	return root, nil
}

// The current version of the schema.
const jsonVersion = 0

func indexJSON(v, i, p interface{}) bool {
	var value interface{}
	var okay = false
	switch object := v.(type) {
	case map[string]interface{}:
		index, ok := i.(string)
		if !ok {
			return false
		}
		value, ok = object[index]
		if !ok {
			return false
		}
		okay = true
	case []interface{}:
		index, ok := i.(int)
		if !ok {
			return false
		}
		if index >= len(object) || index < 0 {
			return false
		}
		value = object[index]
		okay = true
	default:
		return false
	}
	if !okay {
		return false
	}
	switch p := p.(type) {
	case *bool:
		value, ok := value.(bool)
		if !ok {
			return false
		}
		*p = value
	case *float64:
		value, ok := value.(float64)
		if !ok {
			return false
		}
		*p = value
	case *string:
		value, ok := value.(string)
		if !ok {
			return false
		}
		*p = value
	case *[]interface{}:
		value, ok := value.([]interface{})
		if !ok {
			return false
		}
		*p = value
	case *map[string]interface{}:
		value, ok := value.(map[string]interface{})
		if !ok {
			return false
		}
		*p = value
	case *interface{}:
		*p = value
	}
	return true
}

// RootToJSONInterface converts a rbxfile.Root to a generic interface that can
// be read by json.Marshal.
func RootToJSONInterface(root *rbxfile.Root) interface{} {
	refs := rbxfile.References{}
	iroot := make(map[string]interface{}, 2)
	iroot["rbxfile_version"] = float64(jsonVersion)
	instances := make([]interface{}, len(root.Instances))
	for i, inst := range root.Instances {
		instances[i] = InstanceToJSONInterface(inst, refs)
	}
	iroot["instances"] = instances
	return iroot
}

// RootToJSONInterface converts a generic interface produced json.Unmarshal to
// a rbxfile.Root.
func RootFromJSONInterface(iroot interface{}) (root *rbxfile.Root, ok bool) {
	var version float64
	if !indexJSON(iroot, "rbxfile_version", &version) {
		return nil, false
	}

	root = new(rbxfile.Root)

	switch int(version) {
	case 0:
		refs := rbxfile.References{}
		propRefs := []rbxfile.PropRef{}
		root.Instances = make([]*rbxfile.Instance, 0, 8)
		var instances []interface{}
		if !indexJSON(iroot, "instances", &instances) {
			return nil, false
		}
		for _, iinst := range instances {
			inst, ok := InstanceFromJSONInterface(iinst, refs, &propRefs)
			if !ok {
				continue
			}
			root.Instances = append(root.Instances, inst)
		}
		for _, pr := range propRefs {
			pr.Instance.Properties[pr.Property] = rbxfile.ValueReference{
				Instance: refs[pr.Reference],
			}
		}
	default:
		return nil, false
	}
	return root, true
}

////////////////////////////////////////////////////////////////

// InstanceToJSONInterface converts a rbxfile.Instance to a generic interface
// that can be read by json.Marshal.
//
// The refs argument is used by to keep track of instance references.
func InstanceToJSONInterface(inst *rbxfile.Instance, refs rbxfile.References) interface{} {
	iinst := make(map[string]interface{}, 5)
	iinst["class_name"] = inst.ClassName
	iinst["reference"] = refs.Get(inst)
	iinst["is_service"] = inst.IsService
	properties := make(map[string]interface{}, len(inst.Properties))
	for name, prop := range inst.Properties {
		iprop := make(map[string]interface{}, 2)
		iprop["type"] = prop.Type().String()
		iprop["value"] = ValueToJSONInterface(prop, refs)
		properties[name] = iprop
	}
	iinst["properties"] = properties
	children := make([]interface{}, len(inst.Children))
	for i, child := range inst.Children {
		children[i] = InstanceToJSONInterface(child, refs)
	}
	iinst["children"] = children
	return iinst
}

// InstanceFromJSONInterface converts a generic interface produced by
// json.Unmarshal into a rbxfile.Instance.
//
// The refs argument is used to keep track of instance references.
//
// The propRefs argument is populated with a list of PropRefs, specifying
// properties of descendant instances that are references. This should be used
// in combination with refs to set each property after all instance have been
// processed.
func InstanceFromJSONInterface(iinst interface{}, refs rbxfile.References, propRefs *[]rbxfile.PropRef) (inst *rbxfile.Instance, ok bool) {
	var ref string
	indexJSON(iinst, "reference", &ref)
	if rbxfile.IsEmptyReference(ref) {
		inst = rbxfile.NewInstance("", nil)
	} else {
		var exists bool
		if inst, exists = refs[ref]; !exists {
			inst = rbxfile.NewInstance("", nil)
			inst.Reference = ref
			refs[ref] = inst
		}
	}

	if !indexJSON(iinst, "class_name", &inst.ClassName) {
		return nil, false
	}

	indexJSON(iinst, "is_service", &inst.IsService)

	var properties map[string]interface{}
	indexJSON(iinst, "properties", &properties)
	for name, iprop := range properties {
		var typ string
		if !indexJSON(iprop, "type", &typ) {
			continue
		}

		var ivalue interface{}
		if !indexJSON(iprop, "value", &ivalue) {
			continue
		}

		t := rbxfile.TypeFromString(typ)
		value := ValueFromJSONInterface(t, ivalue)
		if value == nil {
			continue
		}
		if t == rbxfile.TypeReference {
			*propRefs = append(*propRefs, rbxfile.PropRef{
				Instance:  inst,
				Property:  name,
				Reference: string(value.(rbxfile.ValueString)),
			})
		} else {
			inst.Properties[name] = value
		}
	}

	var children []interface{}
	indexJSON(iinst, "children", &children)
	inst.Children = make([]*rbxfile.Instance, 0, len(children))
	for _, ichild := range children {
		child, ok := InstanceFromJSONInterface(ichild, refs, propRefs)
		if !ok {
			continue
		}
		child.SetParent(inst)
	}

	return inst, true
}

////////////////////////////////////////////////////////////////

// ValueToJSONInterface converts a value to a generic interface that can be
// read by json.Marshal.
//
// The refs argument is used when converting a rbxfile.ValueReference to a
// string.
func ValueToJSONInterface(value rbxfile.Value, refs rbxfile.References) interface{} {
	switch value := value.(type) {
	case rbxfile.ValueString:
		return string(value)
	case rbxfile.ValueBinaryString:
		var buf bytes.Buffer
		bw := base64.NewEncoder(base64.StdEncoding, &buf)
		bw.Write([]byte(value))
		return buf.String()
	case rbxfile.ValueProtectedString:
		return string(value)
	case rbxfile.ValueContent:
		if len(value) == 0 {
			return nil
		}
		return string(value)
	case rbxfile.ValueBool:
		return bool(value)
	case rbxfile.ValueInt:
		return float64(value)
	case rbxfile.ValueFloat:
		return float64(value)
	case rbxfile.ValueDouble:
		return float64(value)
	case rbxfile.ValueUDim:
		return map[string]interface{}{
			"scale":  float64(value.Scale),
			"offset": float64(value.Offset),
		}
	case rbxfile.ValueUDim2:
		return map[string]interface{}{
			"x": ValueToJSONInterface(value.X, refs),
			"y": ValueToJSONInterface(value.Y, refs),
		}
	case rbxfile.ValueRay:
		return map[string]interface{}{
			"origin":    ValueToJSONInterface(value.Origin, refs),
			"direction": ValueToJSONInterface(value.Direction, refs),
		}
	case rbxfile.ValueFaces:
		return map[string]interface{}{
			"right":  value.Right,
			"top":    value.Top,
			"back":   value.Back,
			"left":   value.Left,
			"bottom": value.Bottom,
			"front":  value.Front,
		}
	case rbxfile.ValueAxes:
		return map[string]interface{}{
			"x": value.X,
			"y": value.Y,
			"z": value.Z,
		}
	case rbxfile.ValueBrickColor:
		return float64(value)
	case rbxfile.ValueColor3:
		return map[string]interface{}{
			"r": float64(value.R),
			"g": float64(value.G),
			"b": float64(value.B),
		}
	case rbxfile.ValueVector2:
		return map[string]interface{}{
			"x": float64(value.X),
			"y": float64(value.Y),
		}
	case rbxfile.ValueVector3:
		return map[string]interface{}{
			"x": float64(value.X),
			"y": float64(value.Y),
			"z": float64(value.Z),
		}
	case rbxfile.ValueCFrame:
		ivalue := make(map[string]interface{}, 2)
		ivalue["position"] = ValueToJSONInterface(value.Position, refs)
		rotation := make([]interface{}, len(value.Rotation))
		for i, r := range value.Rotation {
			rotation[i] = float64(r)
		}
		ivalue["rotation"] = rotation
		return ivalue
	case rbxfile.ValueToken:
		return float64(value)
	case rbxfile.ValueReference:
		return refs.Get(value.Instance)
	case rbxfile.ValueVector3int16:
		return map[string]interface{}{
			"x": float64(value.X),
			"y": float64(value.Y),
			"z": float64(value.Z),
		}
	case rbxfile.ValueVector2int16:
		return map[string]interface{}{
			"x": float64(value.X),
			"y": float64(value.Y),
		}
	case rbxfile.ValueNumberSequence:
		ivalue := make([]interface{}, len(value))
		for i, nsk := range value {
			ivalue[i] = map[string]interface{}{
				"time":     float64(nsk.Time),
				"value":    float64(nsk.Value),
				"envelope": float64(nsk.Envelope),
			}
		}
		return ivalue
	case rbxfile.ValueColorSequence:
		ivalue := make([]interface{}, len(value))
		for i, csk := range value {
			ivalue[i] = map[string]interface{}{
				"time":     float64(csk.Time),
				"value":    ValueToJSONInterface(csk.Value, refs),
				"envelope": float64(csk.Envelope),
			}
		}
		return ivalue
	case rbxfile.ValueNumberRange:
		return map[string]interface{}{
			"min": float64(value.Min),
			"max": float64(value.Max),
		}
	case rbxfile.ValueRect2D:
		return map[string]interface{}{
			"min": ValueToJSONInterface(value.Min, refs),
			"max": ValueToJSONInterface(value.Max, refs),
		}
	case rbxfile.ValuePhysicalProperties:
		return map[string]interface{}{
			"custom_physics":    value.CustomPhysics,
			"density":           float64(value.Density),
			"friction":          float64(value.Friction),
			"elasticity":        float64(value.Elasticity),
			"friction_weight":   float64(value.FrictionWeight),
			"elasticity_weight": float64(value.ElasticityWeight),
		}
	case rbxfile.ValueColor3uint8:
		return map[string]interface{}{
			"r": float64(value.R),
			"g": float64(value.G),
			"b": float64(value.B),
		}
	case rbxfile.ValueInt64:
		return float64(value)
	}
	return nil
}

// ValueFromJSONInterface converts a generic interface produced by
// json.Unmarshal to a rbxfile.Value.
//
// When the value is rbxfile.TypeReference, the result is a
// rbxfile.ValueString containing the raw reference string, expected to be
// dereferenced at a later time.
func ValueFromJSONInterface(typ rbxfile.Type, ivalue interface{}) (value rbxfile.Value) {
	switch typ {
	case rbxfile.TypeString:
		v, ok := ivalue.(string)
		if !ok {
			return nil
		}
		return rbxfile.ValueString(v)
	case rbxfile.TypeBinaryString:
		v, ok := ivalue.(string)
		if !ok {
			return nil
		}
		buf := bytes.NewReader([]byte(v))
		b, err := ioutil.ReadAll(base64.NewDecoder(base64.StdEncoding, buf))
		if err != nil {
			return rbxfile.ValueBinaryString(v)
		}
		return rbxfile.ValueBinaryString(b)
	case rbxfile.TypeProtectedString:
		v, ok := ivalue.(string)
		if !ok {
			return nil
		}
		return rbxfile.ValueProtectedString(v)
	case rbxfile.TypeContent:
		if ivalue == nil {
			return rbxfile.ValueContent(nil)
		}
		v, ok := ivalue.(string)
		if !ok {
			return nil
		}
		return rbxfile.ValueContent(v)
	case rbxfile.TypeBool:
		v, ok := ivalue.(bool)
		if !ok {
			return nil
		}
		return rbxfile.ValueBool(v)
	case rbxfile.TypeInt:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return rbxfile.ValueInt(int32(v))
	case rbxfile.TypeFloat:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return rbxfile.ValueFloat(float32(v))
	case rbxfile.TypeDouble:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return rbxfile.ValueDouble(v)
	case rbxfile.TypeUDim:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueUDim{
			Scale:  v["scale"].(float32),
			Offset: v["offset"].(int16),
		}
	case rbxfile.TypeUDim2:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueUDim2{
			X: ValueFromJSONInterface(rbxfile.TypeUDim, v["x"]).(rbxfile.ValueUDim),
			Y: ValueFromJSONInterface(rbxfile.TypeUDim, v["y"]).(rbxfile.ValueUDim),
		}
	case rbxfile.TypeRay:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueRay{
			Origin:    ValueFromJSONInterface(rbxfile.TypeVector3, v["origin"]).(rbxfile.ValueVector3),
			Direction: ValueFromJSONInterface(rbxfile.TypeVector3, v["direction"]).(rbxfile.ValueVector3),
		}
	case rbxfile.TypeFaces:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueFaces{
			Right:  v["right"].(bool),
			Top:    v["top"].(bool),
			Back:   v["back"].(bool),
			Left:   v["left"].(bool),
			Bottom: v["bottom"].(bool),
			Front:  v["front"].(bool),
		}
	case rbxfile.TypeAxes:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueAxes{
			X: v["x"].(bool),
			Y: v["y"].(bool),
			Z: v["z"].(bool),
		}
	case rbxfile.TypeBrickColor:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return rbxfile.ValueBrickColor(uint32(v))
	case rbxfile.TypeColor3:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueColor3{
			R: float32(v["r"].(float64)),
			G: float32(v["g"].(float64)),
			B: float32(v["b"].(float64)),
		}
	case rbxfile.TypeVector2:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueVector2{
			X: float32(v["x"].(float64)),
			Y: float32(v["y"].(float64)),
		}
	case rbxfile.TypeVector3:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueVector3{
			X: float32(v["x"].(float64)),
			Y: float32(v["y"].(float64)),
			Z: float32(v["z"].(float64)),
		}
	case rbxfile.TypeCFrame:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		value := rbxfile.ValueCFrame{
			Position: ValueFromJSONInterface(rbxfile.TypeVector3, v["position"]).(rbxfile.ValueVector3),
		}
		irotation, ok := v["rotation"].([]interface{})
		if !ok {
			return value
		}
		for i, irot := range irotation {
			if i >= len(value.Rotation) {
				break
			}
			value.Rotation[i] = float32(irot.(float64))
		}
		return value
	case rbxfile.TypeToken:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return rbxfile.ValueToken(uint32(v))
	case rbxfile.TypeReference:
		v, ok := ivalue.(string)
		if !ok {
			return nil
		}
		// ValueReference is handled as a special case, so return as a
		// ValueString.
		return rbxfile.ValueString(v)
	case rbxfile.TypeVector3int16:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueVector3int16{
			X: int16(v["x"].(float64)),
			Y: int16(v["y"].(float64)),
			Z: int16(v["z"].(float64)),
		}
	case rbxfile.TypeVector2int16:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueVector2int16{
			X: int16(v["x"].(float64)),
			Y: int16(v["y"].(float64)),
		}
	case rbxfile.TypeNumberSequence:
		v, ok := ivalue.([]interface{})
		if !ok {
			return nil
		}
		value := make(rbxfile.ValueNumberSequence, len(v))
		for i, insk := range v {
			insk, ok := insk.(map[string]interface{})
			if !ok {
				continue
			}
			value[i] = rbxfile.ValueNumberSequenceKeypoint{
				Time:     float32(insk["time"].(float64)),
				Value:    float32(insk["value"].(float64)),
				Envelope: float32(insk["envelope"].(float64)),
			}
		}
		return value
	case rbxfile.TypeColorSequence:
		v, ok := ivalue.([]interface{})
		if !ok {
			return nil
		}
		value := make(rbxfile.ValueColorSequence, len(v))
		for i, icsk := range v {
			icsk, ok := icsk.(map[string]interface{})
			if !ok {
				continue
			}
			value[i] = rbxfile.ValueColorSequenceKeypoint{
				Time:     float32(icsk["time"].(float64)),
				Value:    ValueFromJSONInterface(rbxfile.TypeColor3, icsk["value"]).(rbxfile.ValueColor3),
				Envelope: float32(icsk["envelope"].(float64)),
			}
		}
		return value
	case rbxfile.TypeNumberRange:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueNumberRange{
			Min: float32(v["min"].(float64)),
			Max: float32(v["max"].(float64)),
		}
	case rbxfile.TypeRect2D:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueRect2D{
			Min: ValueFromJSONInterface(rbxfile.TypeVector2, v["min"]).(rbxfile.ValueVector2),
			Max: ValueFromJSONInterface(rbxfile.TypeVector2, v["max"]).(rbxfile.ValueVector2),
		}
	case rbxfile.TypePhysicalProperties:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValuePhysicalProperties{
			CustomPhysics:    v["custom_physics"].(bool),
			Density:          float32(v["density"].(float64)),
			Friction:         float32(v["friction"].(float64)),
			Elasticity:       float32(v["elasticity"].(float64)),
			FrictionWeight:   float32(v["friction_weight"].(float64)),
			ElasticityWeight: float32(v["elasticity_weight"].(float64)),
		}
	case rbxfile.TypeColor3uint8:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return rbxfile.ValueColor3uint8{
			R: byte(v["r"].(float64)),
			G: byte(v["g"].(float64)),
			B: byte(v["b"].(float64)),
		}
	case rbxfile.TypeInt64:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return rbxfile.ValueInt64(int64(v))
	}
	return nil
}
