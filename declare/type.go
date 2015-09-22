package declare

import (
	"github.com/robloxapi/rbxfile"
)

// Type corresponds to a rbxfile.Type.
type Type byte

const (
	_ Type = iota
	String
	BinaryString
	ProtectedString
	Content
	Bool
	Int
	Float
	Double
	UDim
	UDim2
	Ray
	Faces
	Axes
	BrickColor
	Color3
	Vector2
	Vector3
	CFrame
	Token
	Reference
	Vector3int16
	Vector2int16
)

func normInt16(v interface{}) int16 {
	switch v := v.(type) {
	case int:
		return int16(v)
	case uint:
		return int16(v)
	case uint8:
		return int16(v)
	case uint16:
		return int16(v)
	case uint32:
		return int16(v)
	case uint64:
		return int16(v)
	case int8:
		return int16(v)
	case int16:
		return int16(v)
	case int32:
		return int16(v)
	case int64:
		return int16(v)
	case float32:
		return int16(v)
	case float64:
		return int16(v)
	}

	return 0
}

func normInt32(v interface{}) int32 {
	switch v := v.(type) {
	case int:
		return int32(v)
	case uint:
		return int32(v)
	case uint8:
		return int32(v)
	case uint16:
		return int32(v)
	case uint32:
		return int32(v)
	case uint64:
		return int32(v)
	case int8:
		return int32(v)
	case int16:
		return int32(v)
	case int32:
		return int32(v)
	case int64:
		return int32(v)
	case float32:
		return int32(v)
	case float64:
		return int32(v)
	}

	return 0
}

func normUint32(v interface{}) uint32 {
	switch v := v.(type) {
	case int:
		return uint32(v)
	case uint:
		return uint32(v)
	case uint8:
		return uint32(v)
	case uint16:
		return uint32(v)
	case uint32:
		return uint32(v)
	case uint64:
		return uint32(v)
	case int8:
		return uint32(v)
	case int16:
		return uint32(v)
	case int32:
		return uint32(v)
	case int64:
		return uint32(v)
	case float32:
		return uint32(v)
	case float64:
		return uint32(v)
	}

	return 0
}

func normFloat32(v interface{}) float32 {
	switch v := v.(type) {
	case int:
		return float32(v)
	case uint:
		return float32(v)
	case uint8:
		return float32(v)
	case uint16:
		return float32(v)
	case uint32:
		return float32(v)
	case uint64:
		return float32(v)
	case int8:
		return float32(v)
	case int16:
		return float32(v)
	case int32:
		return float32(v)
	case int64:
		return float32(v)
	case float32:
		return float32(v)
	case float64:
		return float32(v)
	}

	return 0
}

func normFloat64(v interface{}) float64 {
	switch v := v.(type) {
	case int:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return float64(v)
	}

	return 0
}

func normBool(v interface{}) bool {
	vv, _ := v.(bool)
	return vv
}

func assertValue(t Type, v interface{}) (value rbxfile.Value, ok bool) {
	switch t {
	case String:
		value, ok = v.(rbxfile.ValueString)
	case BinaryString:
		value, ok = v.(rbxfile.ValueBinaryString)
	case ProtectedString:
		value, ok = v.(rbxfile.ValueProtectedString)
	case Content:
		value, ok = v.(rbxfile.ValueContent)
	case Bool:
		value, ok = v.(rbxfile.ValueBool)
	case Int:
		value, ok = v.(rbxfile.ValueInt)
	case Float:
		value, ok = v.(rbxfile.ValueFloat)
	case Double:
		value, ok = v.(rbxfile.ValueDouble)
	case UDim:
		value, ok = v.(rbxfile.ValueUDim)
	case UDim2:
		value, ok = v.(rbxfile.ValueUDim2)
	case Ray:
		value, ok = v.(rbxfile.ValueRay)
	case Faces:
		value, ok = v.(rbxfile.ValueFaces)
	case Axes:
		value, ok = v.(rbxfile.ValueAxes)
	case BrickColor:
		value, ok = v.(rbxfile.ValueBrickColor)
	case Color3:
		value, ok = v.(rbxfile.ValueColor3)
	case Vector2:
		value, ok = v.(rbxfile.ValueVector2)
	case Vector3:
		value, ok = v.(rbxfile.ValueVector3)
	case CFrame:
		value, ok = v.(rbxfile.ValueCFrame)
	case Token:
		value, ok = v.(rbxfile.ValueToken)
	case Reference:
		value, ok = v.(rbxfile.ValueReference)
	case Vector3int16:
		value, ok = v.(rbxfile.ValueVector3int16)
	case Vector2int16:
		value, ok = v.(rbxfile.ValueVector2int16)
	}
	return
}

func (t Type) value(refs map[string]*rbxfile.Instance, v []interface{}) rbxfile.Value {
	if len(v) == 0 {
		goto zero
	}

	if v, ok := assertValue(t, v[0]); ok {
		return v
	}

	switch t {
	case String:
		switch v := v[0].(type) {
		case string:
			return rbxfile.ValueString(v)
		case []byte:
			return rbxfile.ValueString(v)
		}
	case BinaryString:
		switch v := v[0].(type) {
		case string:
			return rbxfile.ValueString(v)
		case []byte:
			return rbxfile.ValueString(v)
		}
	case ProtectedString:
		switch v := v[0].(type) {
		case string:
			return rbxfile.ValueString(v)
		case []byte:
			return rbxfile.ValueString(v)
		}
	case Content:
		switch v := v[0].(type) {
		case string:
			return rbxfile.ValueString(v)
		case []byte:
			return rbxfile.ValueString(v)
		}
	case Bool:
		switch v := v[0].(type) {
		case bool:
			return rbxfile.ValueBool(v)
		}
	case Int:
		return rbxfile.ValueInt(normInt32(v[0]))
	case Float:
		return rbxfile.ValueFloat(normFloat32(v[0]))
	case Double:
		return rbxfile.ValueFloat(normFloat64(v[0]))
	case UDim:
		if len(v) == 2 {
			return rbxfile.ValueUDim{
				Scale:  normFloat32(v[0]),
				Offset: normInt16(v[1]),
			}
		}
	case UDim2:
		switch len(v) {
		case 2:
			x, _ := v[0].(rbxfile.ValueUDim)
			y, _ := v[1].(rbxfile.ValueUDim)
			return rbxfile.ValueUDim2{
				X: x,
				Y: y,
			}
		case 4:
			return rbxfile.ValueUDim2{
				X: rbxfile.ValueUDim{
					Scale:  normFloat32(v[0]),
					Offset: normInt16(v[1]),
				},
				Y: rbxfile.ValueUDim{
					Scale:  normFloat32(v[2]),
					Offset: normInt16(v[3]),
				},
			}
		}
	case Ray:
		switch len(v) {
		case 2:
			origin, _ := v[0].(rbxfile.ValueVector3)
			direction, _ := v[1].(rbxfile.ValueVector3)
			return rbxfile.ValueRay{
				Origin:    origin,
				Direction: direction,
			}
		case 6:
			return rbxfile.ValueRay{
				Origin: rbxfile.ValueVector3{
					X: normFloat32(v[0]),
					Y: normFloat32(v[1]),
					Z: normFloat32(v[2]),
				},
				Direction: rbxfile.ValueVector3{
					X: normFloat32(v[3]),
					Y: normFloat32(v[4]),
					Z: normFloat32(v[5]),
				},
			}
		}
	case Faces:
		if len(v) == 6 {
			return rbxfile.ValueFaces{
				Right:  normBool(v[0]),
				Top:    normBool(v[1]),
				Back:   normBool(v[2]),
				Left:   normBool(v[3]),
				Bottom: normBool(v[4]),
				Front:  normBool(v[5]),
			}
		}
	case Axes:
		if len(v) == 3 {
			return rbxfile.ValueAxes{
				X: normBool(v[0]),
				Y: normBool(v[1]),
				Z: normBool(v[2]),
			}
		}
	case BrickColor:
		return rbxfile.ValueBrickColor(normUint32(v[0]))
	case Color3:
		if len(v) == 3 {
			return rbxfile.ValueColor3{
				R: normFloat32(v[0]),
				G: normFloat32(v[1]),
				B: normFloat32(v[2]),
			}
		}
	case Vector2:
		if len(v) == 2 {
			return rbxfile.ValueVector2{
				X: normFloat32(v[0]),
				Y: normFloat32(v[1]),
			}
		}
	case Vector3:
		if len(v) == 3 {
			return rbxfile.ValueVector3{
				X: normFloat32(v[0]),
				Y: normFloat32(v[1]),
				Z: normFloat32(v[2]),
			}
		}
	case CFrame:
		switch len(v) {
		case 10:
			p, _ := v[0].(rbxfile.ValueVector3)
			return rbxfile.ValueCFrame{
				Position: p,
				Rotation: [9]float32{
					normFloat32(v[0]),
					normFloat32(v[1]),
					normFloat32(v[2]),
					normFloat32(v[3]),
					normFloat32(v[4]),
					normFloat32(v[5]),
					normFloat32(v[6]),
					normFloat32(v[7]),
					normFloat32(v[8]),
				},
			}
		case 12:
			return rbxfile.ValueCFrame{
				Position: rbxfile.ValueVector3{
					normFloat32(v[0]),
					normFloat32(v[1]),
					normFloat32(v[2]),
				},
				Rotation: [9]float32{
					normFloat32(v[3]),
					normFloat32(v[4]),
					normFloat32(v[5]),
					normFloat32(v[6]),
					normFloat32(v[7]),
					normFloat32(v[8]),
					normFloat32(v[9]),
					normFloat32(v[10]),
					normFloat32(v[11]),
				},
			}
		}
	case Token:
		return rbxfile.ValueToken(normUint32(v[0]))
	case Reference:
		switch v := v[0].(type) {
		case string:
			return rbxfile.ValueReference{
				Instance: refs[v],
			}
		case []byte:
			return rbxfile.ValueReference{
				Instance: refs[string(v)],
			}
		case *rbxfile.Instance:
			return rbxfile.ValueReference{
				Instance: v,
			}
		}
	case Vector3int16:
		if len(v) == 3 {
			return rbxfile.ValueVector3int16{
				X: normInt16(v[0]),
				Y: normInt16(v[1]),
				Z: normInt16(v[2]),
			}
		}
	case Vector2int16:
		if len(v) == 2 {
			return rbxfile.ValueVector2int16{
				X: normInt16(v[0]),
				Y: normInt16(v[1]),
			}
		}
	}

zero:
	return rbxfile.NewValue(rbxfile.Type(t))
}
