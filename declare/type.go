package declare

import (
	"github.com/robloxapi/rbxfile"
	"strings"
)

// Type corresponds to a rbxfile.Type.
type Type byte

// String returns a string representation of the type. If the type is not
// valid, then the returned value will be "Invalid".
func (t Type) String() string {
	s, ok := typeStrings[t]
	if !ok {
		return "Invalid"
	}
	return s
}

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
	NumberSequence
	ColorSequence
	NumberRange
	Rect2D
	PhysicalProperties
	Color3uint8
)

// TypeFromString returns a Type from its string representation. Type(0) is
// returned if the string does not represent an existing Type.
func TypeFromString(s string) Type {
	s = strings.ToLower(s)
	for typ, str := range typeStrings {
		if s == strings.ToLower(str) {
			return typ
		}
	}
	return 0
}

var typeStrings = map[Type]string{
	String:             "String",
	BinaryString:       "BinaryString",
	ProtectedString:    "ProtectedString",
	Content:            "Content",
	Bool:               "Bool",
	Int:                "Int",
	Float:              "Float",
	Double:             "Double",
	UDim:               "UDim",
	UDim2:              "UDim2",
	Ray:                "Ray",
	Faces:              "Faces",
	Axes:               "Axes",
	BrickColor:         "BrickColor",
	Color3:             "Color3",
	Vector2:            "Vector2",
	Vector3:            "Vector3",
	CFrame:             "CFrame",
	Token:              "Token",
	Reference:          "Reference",
	Vector3int16:       "Vector3int16",
	Vector2int16:       "Vector2int16",
	NumberSequence:     "NumberSequence",
	ColorSequence:      "ColorSequence",
	NumberRange:        "NumberRange",
	Rect2D:             "Rect2D",
	PhysicalProperties: "PhysicalProperties",
	Color3uint8:        "Color3uint8",
}

func normUint8(v interface{}) uint8 {
	switch v := v.(type) {
	case int:
		return uint8(v)
	case uint:
		return uint8(v)
	case uint8:
		return uint8(v)
	case uint16:
		return uint8(v)
	case uint32:
		return uint8(v)
	case uint64:
		return uint8(v)
	case int8:
		return uint8(v)
	case int16:
		return uint8(v)
	case int32:
		return uint8(v)
	case int64:
		return uint8(v)
	case float32:
		return uint8(v)
	case float64:
		return uint8(v)
	}

	return 0
}

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
	case NumberSequence:
		value, ok = v.(rbxfile.ValueNumberSequence)
	case ColorSequence:
		value, ok = v.(rbxfile.ValueColorSequence)
	case NumberRange:
		value, ok = v.(rbxfile.ValueNumberRange)
	case Rect2D:
		value, ok = v.(rbxfile.ValueRect2D)
	case PhysicalProperties:
		value, ok = v.(rbxfile.ValuePhysicalProperties)
	case Color3uint8:
		value, ok = v.(rbxfile.ValueColor3uint8)
	}
	return
}

func (t Type) value(refs rbxfile.References, v []interface{}) rbxfile.Value {
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
	case NumberSequence:
		if len(v) > 0 {
			if _, ok := v[0].(rbxfile.ValueNumberSequenceKeypoint); ok && len(v) >= 2 {
				ns := make(rbxfile.ValueNumberSequence, len(v))
				for i, k := range v {
					k, _ := k.(rbxfile.ValueNumberSequenceKeypoint)
					ns[i] = k
				}
				return ns
			} else if len(v)%3 == 0 && len(v) >= 6 {
				ns := make(rbxfile.ValueNumberSequence, len(v)/3)
				for i := 0; i < len(v); i += 3 {
					ns[i/3] = rbxfile.ValueNumberSequenceKeypoint{
						Time:     normFloat32(v[i+0]),
						Value:    normFloat32(v[i+1]),
						Envelope: normFloat32(v[i+2]),
					}
				}
			}
		}
	case ColorSequence:
		if len(v) > 0 {
			if _, ok := v[0].(rbxfile.ValueColorSequenceKeypoint); ok && len(v) >= 2 {
				cs := make(rbxfile.ValueColorSequence, len(v))
				for i, k := range v {
					k, _ := k.(rbxfile.ValueColorSequenceKeypoint)
					cs[i] = k
				}
				return cs
			} else if _, ok := v[1].(rbxfile.ValueColor3); ok && len(v)%3 == 0 && len(v) >= 6 {
				cs := make(rbxfile.ValueColorSequence, len(v)/3)
				for i := 0; i < len(v); i += 3 {
					kval, _ := v[i+1].(rbxfile.ValueColor3)
					cs[i/3] = rbxfile.ValueColorSequenceKeypoint{
						Time:     normFloat32(v[i+0]),
						Value:    kval,
						Envelope: normFloat32(v[i+2]),
					}
				}
			} else if len(v)%5 == 0 && len(v) >= 10 {
				cs := make(rbxfile.ValueColorSequence, len(v)/5)
				for i := 0; i < len(v); i += 5 {
					cs[i/5] = rbxfile.ValueColorSequenceKeypoint{
						Time: normFloat32(v[i+0]),
						Value: rbxfile.ValueColor3{
							R: normFloat32(v[i+1]),
							G: normFloat32(v[i+2]),
							B: normFloat32(v[i+3]),
						},
						Envelope: normFloat32(v[i+4]),
					}
				}
			}
		}
	case NumberRange:
		if len(v) == 2 {
			return rbxfile.ValueNumberRange{
				Min: normFloat32(v[0]),
				Max: normFloat32(v[1]),
			}
		}
	case Rect2D:
		switch len(v) {
		case 2:
			min, _ := v[0].(rbxfile.ValueVector2)
			max, _ := v[0].(rbxfile.ValueVector2)
			return rbxfile.ValueRect2D{
				Min: min,
				Max: max,
			}
		case 4:
			return rbxfile.ValueRect2D{
				Min: rbxfile.ValueVector2{
					X: normFloat32(v[0]),
					Y: normFloat32(v[1]),
				},
				Max: rbxfile.ValueVector2{
					X: normFloat32(v[2]),
					Y: normFloat32(v[3]),
				},
			}
		}
	case PhysicalProperties:
		switch len(v) {
		case 0:
			return rbxfile.ValuePhysicalProperties{}
		case 3:
			return rbxfile.ValuePhysicalProperties{
				CustomPhysics: true,
				Density:       normFloat32(v[0]),
				Friction:      normFloat32(v[1]),
				Elasticity:    normFloat32(v[2]),
			}
		case 5:
			return rbxfile.ValuePhysicalProperties{
				CustomPhysics:    true,
				Density:          normFloat32(v[0]),
				Friction:         normFloat32(v[1]),
				Elasticity:       normFloat32(v[2]),
				FrictionWeight:   normFloat32(v[3]),
				ElasticityWeight: normFloat32(v[4]),
			}
		}
	case Color3uint8:
		if len(v) == 3 {
			return rbxfile.ValueColor3uint8{
				R: normUint8(v[0]),
				G: normUint8(v[1]),
				B: normUint8(v[2]),
			}
		}
	}

zero:
	return rbxfile.NewValue(rbxfile.Type(t))
}
