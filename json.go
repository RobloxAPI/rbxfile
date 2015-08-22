package rbxfile

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
)

func (root *Root) MarshalJSON() (b []byte, err error) {
	return json.Marshal(rootToJSONInterface(root))
}

func (root *Root) UnmarshalJSON(b []byte) (err error) {
	var v interface{}
	err = json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	r, ok := rootFromJSONInterface(v)
	if !ok {
		return errors.New("invalid JSON Root object")
	}
	*root = *r
	return nil
}

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

func rootToJSONInterface(root *Root) interface{} {
	refs := map[string]*Instance{}
	iroot := make(map[string]interface{}, 2)
	iroot["rbxfile_version"] = float64(jsonVersion)
	instances := make([]interface{}, len(root.Instances))
	for i, inst := range root.Instances {
		instances[i] = instanceToJSONInterface(inst, refs)
	}
	iroot["instances"] = instances
	return iroot
}

type propRef struct {
	inst *Instance
	prop string
	ref  string
}

func rootFromJSONInterface(iroot interface{}) (root *Root, ok bool) {
	var version float64
	if !indexJSON(iroot, "rbxfile_version", &version) {
		return nil, false
	}

	root = new(Root)

	switch int(version) {
	case 0:
		refs := map[string]*Instance{}
		propRefs := []propRef{}
		root.Instances = make([]*Instance, 0, 8)
		var instances []interface{}
		if !indexJSON(iroot, "instances", &instances) {
			return nil, false
		}
		for _, iinst := range instances {
			inst, ok := instanceFromJSONInterface(iinst, refs, &propRefs)
			if !ok {
				continue
			}
			root.Instances = append(root.Instances, inst)
		}
		for _, propRef := range propRefs {
			propRef.inst.Properties[propRef.prop] = ValueReference{
				Instance: refs[propRef.ref],
			}
		}
	default:
		return nil, false
	}
	return root, true
}

////////////////////////////////////////////////////////////////

func instanceToJSONInterface(inst *Instance, refs map[string]*Instance) interface{} {
	iinst := make(map[string]interface{}, 5)
	iinst["class_name"] = inst.ClassName
	iinst["reference"] = GetReference(inst, refs)
	iinst["is_service"] = inst.IsService
	properties := make(map[string]interface{}, len(inst.Properties))
	for name, prop := range inst.Properties {
		iprop := make(map[string]interface{}, 2)
		iprop["type"] = prop.Type().String()
		iprop["value"] = valueToJSONInterface(prop, refs)
		properties[name] = iprop
	}
	iinst["properties"] = properties
	children := make([]interface{}, len(inst.children))
	for i, child := range inst.GetChildren() {
		children[i] = instanceToJSONInterface(child, refs)
	}
	iinst["children"] = children
	return iinst
}

func instanceFromJSONInterface(iinst interface{}, refs map[string]*Instance, propRefs *[]propRef) (inst *Instance, ok bool) {
	inst = new(Instance)

	if !indexJSON(iinst, "class_name", &inst.ClassName) {
		return nil, false
	}
	var ref string
	if !indexJSON(iinst, "reference", &ref) {
		return inst, true
	}
	inst.Reference = []byte(ref)
	refs[ref] = inst
	if !indexJSON(iinst, "is_service", &inst.IsService) {
		return inst, true
	}

	var properties map[string]interface{}
	if !indexJSON(iinst, "properies", &properties) {
		return inst, true
	}

	for name, iprop := range properties {
		var typ string
		if !indexJSON(iprop, "type", &typ) {
			continue
		}

		var ivalue interface{}
		if !indexJSON(iprop, "value", &ivalue) {
			continue
		}

		t := TypeFromString(typ)
		value := valueFromJSONInterface(t, ivalue)
		if value == nil {
			continue
		}
		if t == TypeReference {
			*propRefs = append(*propRefs, propRef{
				inst: inst,
				prop: name,
				ref:  string(value.(ValueString)),
			})
		} else {
			inst.Properties[name] = value
		}
	}

	var children []interface{}
	if !indexJSON(iinst, "children", &children) {
		return inst, true
	}

	inst.children = make([]*Instance, len(children))
	for _, ichild := range children {
		child, ok := instanceFromJSONInterface(ichild, refs, propRefs)
		if !ok {
			continue
		}
		child.SetParent(inst)
	}
	return inst, true
}

////////////////////////////////////////////////////////////////

func valueToJSONInterface(value Value, refs map[string]*Instance) interface{} {
	switch value := value.(type) {
	case ValueString:
		return string(value)
	case ValueBinaryString:
		var buf bytes.Buffer
		bw := base64.NewEncoder(base64.StdEncoding, &buf)
		bw.Write([]byte(value))
		return buf.String()
	case ValueProtectedString:
		return string(value)
	case ValueContent:
		if len(value) == 0 {
			return nil
		}
		return string(value)
	case ValueBool:
		return bool(value)
	case ValueInt:
		return float64(value)
	case ValueFloat:
		return float64(value)
	case ValueDouble:
		return float64(value)
	case ValueUDim:
		return map[string]interface{}{
			"scale":  float64(value.Scale),
			"offset": float64(value.Offset),
		}
	case ValueUDim2:
		return map[string]interface{}{
			"x": valueToJSONInterface(value.X, refs),
			"y": valueToJSONInterface(value.Y, refs),
		}
	case ValueRay:
		return map[string]interface{}{
			"origin":    valueToJSONInterface(value.Origin, refs),
			"direction": valueToJSONInterface(value.Direction, refs),
		}
	case ValueFaces:
		return map[string]interface{}{
			"right":  value.Right,
			"top":    value.Top,
			"back":   value.Back,
			"left":   value.Left,
			"bottom": value.Bottom,
			"front":  value.Front,
		}
	case ValueAxes:
		return map[string]interface{}{
			"x": value.X,
			"y": value.Y,
			"z": value.Z,
		}
	case ValueBrickColor:
		return float64(value)
	case ValueColor3:
		return map[string]interface{}{
			"r": float64(value.R),
			"g": float64(value.G),
			"b": float64(value.B),
		}
	case ValueVector2:
		return map[string]interface{}{
			"x": float64(value.X),
			"y": float64(value.Y),
		}
	case ValueVector3:
		return map[string]interface{}{
			"x": float64(value.X),
			"y": float64(value.Y),
			"z": float64(value.Z),
		}
	case ValueCFrame:
		ivalue := make(map[string]interface{}, 2)
		ivalue["position"] = valueToJSONInterface(value.Position, refs)
		rotation := make([]interface{}, len(value.Rotation))
		for i, r := range value.Rotation {
			rotation[i] = float64(r)
		}
		ivalue["rotation"] = rotation
		return ivalue
	case ValueToken:
		return float64(value)
	case ValueReference:
		return GetReference(value.Instance, refs)
	case ValueVector3int16:
		return map[string]interface{}{
			"x": float64(value.X),
			"y": float64(value.Y),
			"z": float64(value.Z),
		}
	case ValueVector2int16:
		return map[string]interface{}{
			"x": float64(value.X),
			"y": float64(value.Y),
		}
	case ValueNumberSequence:
		ivalue := make([]interface{}, len(value))
		for i, nsk := range value {
			ivalue[i] = map[string]interface{}{
				"time":     float64(nsk.Time),
				"value":    float64(nsk.Value),
				"envelope": float64(nsk.Envelope),
			}
		}
		return ivalue
	case ValueColorSequence:
		ivalue := make([]interface{}, len(value))
		for i, csk := range value {
			ivalue[i] = map[string]interface{}{
				"time":     float64(csk.Time),
				"value":    valueToJSONInterface(csk.Value, refs),
				"envelope": float64(csk.Envelope),
			}
		}
		return ivalue
	case ValueNumberRange:
		return map[string]interface{}{
			"min": float64(value.Min),
			"max": float64(value.Max),
		}
	case ValueRect2D:
		return map[string]interface{}{
			"min": valueToJSONInterface(value.Min, refs),
			"max": valueToJSONInterface(value.Max, refs),
		}
	}
	return nil
}

func valueFromJSONInterface(typ Type, ivalue interface{}) (value Value) {
	switch typ {
	case TypeString:
		v, ok := ivalue.(string)
		if !ok {
			return nil
		}
		return ValueString(v)
	case TypeBinaryString:
		v, ok := ivalue.(string)
		if !ok {
			return nil
		}
		buf := bytes.NewReader([]byte(v))
		b, err := ioutil.ReadAll(base64.NewDecoder(base64.StdEncoding, buf))
		if err != nil {
			return ValueBinaryString(v)
		}
		return ValueBinaryString(b)
	case TypeProtectedString:
		v, ok := ivalue.(string)
		if !ok {
			return nil
		}
		return ValueProtectedString(v)
	case TypeContent:
		if ivalue == nil {
			return ValueContent(nil)
		}
		v, ok := ivalue.(string)
		if !ok {
			return nil
		}
		return ValueContent(v)
	case TypeBool:
		v, ok := ivalue.(bool)
		if !ok {
			return nil
		}
		return ValueBool(v)
	case TypeInt:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return ValueInt(int32(v))
	case TypeFloat:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return ValueInt(float32(v))
	case TypeDouble:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return ValueInt(v)
	case TypeUDim:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueUDim{
			Scale:  v["scale"].(float32),
			Offset: v["offset"].(int16),
		}
	case TypeUDim2:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueUDim2{
			X: valueFromJSONInterface(TypeUDim, v["x"]).(ValueUDim),
			Y: valueFromJSONInterface(TypeUDim, v["y"]).(ValueUDim),
		}
	case TypeRay:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueRay{
			Origin:    valueFromJSONInterface(TypeVector3, v["origin"]).(ValueVector3),
			Direction: valueFromJSONInterface(TypeVector3, v["direction"]).(ValueVector3),
		}
	case TypeFaces:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueFaces{
			Right:  v["right"].(bool),
			Top:    v["top"].(bool),
			Back:   v["back"].(bool),
			Left:   v["left"].(bool),
			Bottom: v["bottom"].(bool),
			Front:  v["front"].(bool),
		}
	case TypeAxes:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueAxes{
			X: v["x"].(bool),
			Y: v["y"].(bool),
			Z: v["z"].(bool),
		}
	case TypeBrickColor:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return ValueInt(uint32(v))
	case TypeColor3:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueColor3{
			R: float32(v["r"].(float64)),
			G: float32(v["g"].(float64)),
			B: float32(v["b"].(float64)),
		}
	case TypeVector2:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueVector2{
			X: float32(v["x"].(float64)),
			Y: float32(v["y"].(float64)),
		}
	case TypeVector3:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueVector3{
			X: float32(v["x"].(float64)),
			Y: float32(v["y"].(float64)),
			Z: float32(v["z"].(float64)),
		}
	case TypeCFrame:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		value := ValueCFrame{
			Position: valueFromJSONInterface(TypeVector3, v["position"]).(ValueVector3),
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
	case TypeToken:
		v, ok := ivalue.(float64)
		if !ok {
			return nil
		}
		return ValueInt(uint32(v))
	case TypeReference:
		v, ok := ivalue.(string)
		if !ok {
			return nil
		}
		// ValueReference is handled as a special case, so return as a
		// ValueString.
		return ValueString(v)
	case TypeVector3int16:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueVector3int16{
			X: int16(v["x"].(float64)),
			Y: int16(v["y"].(float64)),
			Z: int16(v["z"].(float64)),
		}
	case TypeVector2int16:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueVector2int16{
			X: int16(v["x"].(float64)),
			Y: int16(v["y"].(float64)),
		}
	case TypeNumberSequence:
		v, ok := ivalue.([]interface{})
		if !ok {
			return nil
		}
		value := make(ValueNumberSequence, len(v))
		for i, insk := range v {
			insk, ok := insk.(map[string]interface{})
			if !ok {
				continue
			}
			value[i] = ValueNumberSequenceKeypoint{
				Time:     float32(insk["time"].(float64)),
				Value:    float32(insk["value"].(float64)),
				Envelope: float32(insk["envelope"].(float64)),
			}
		}
		return value
	case TypeColorSequence:
		v, ok := ivalue.([]interface{})
		if !ok {
			return nil
		}
		value := make(ValueColorSequence, len(v))
		for i, icsk := range v {
			icsk, ok := icsk.(map[string]interface{})
			if !ok {
				continue
			}
			value[i] = ValueColorSequenceKeypoint{
				Time:     float32(icsk["time"].(float64)),
				Value:    valueFromJSONInterface(TypeColor3, icsk["value"]).(ValueColor3),
				Envelope: float32(icsk["envelope"].(float64)),
			}
		}
		return value
	case TypeNumberRange:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueNumberRange{
			Min: float32(v["min"].(float64)),
			Max: float32(v["max"].(float64)),
		}
	case TypeRect2D:
		v, ok := ivalue.(map[string]interface{})
		if !ok {
			return nil
		}
		return ValueRect2D{
			Min: valueFromJSONInterface(TypeVector2, v["min"]).(ValueVector2),
			Max: valueFromJSONInterface(TypeVector2, v["max"]).(ValueVector2),
		}
	}
	return nil
}
