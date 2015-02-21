package rbxfile

import (
	"strconv"
	"strings"
)

// Type represents a Roblox type.
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
	TypeInvalid Type = iota
	TypeString
	TypeBinaryString
	TypeProtectedString
	TypeContent
	TypeBool
	TypeInt
	TypeFloat
	TypeDouble
	TypeUDim
	TypeUDim2
	TypeRay
	TypeFaces
	TypeAxes
	TypeBrickColor
	TypeColor3
	TypeVector2
	TypeVector3
	TypeCFrame
	TypeToken
	TypeReference
	TypeVector3int16
	TypeVector2int16
	TypeRegion3
	TypeRegion3int16
)

var typeStrings = map[Type]string{
	TypeString:          "string",
	TypeBinaryString:    "BinaryString",
	TypeProtectedString: "ProtectedString",
	TypeContent:         "Content",
	TypeBool:            "bool",
	TypeInt:             "int",
	TypeFloat:           "float",
	TypeDouble:          "double",
	TypeUDim:            "UDim",
	TypeUDim2:           "UDim2",
	TypeRay:             "Ray",
	TypeFaces:           "Faces",
	TypeAxes:            "Axes",
	TypeBrickColor:      "BrickColor",
	TypeColor3:          "Color3",
	TypeVector2:         "Vector2",
	TypeVector3:         "Vector3",
	TypeCFrame:          "CoordinateFrame",
	TypeToken:           "token",
	TypeReference:       "Ref",
	TypeVector3int16:    "Vector3int16",
	TypeVector2int16:    "Vector2int16",
	TypeRegion3:         "Region3",
	TypeRegion3int16:    "Region3int16",
}

// Value holds a value of a particular Type.
type Value interface {
	// TypeString returns the name of the type.
	Type() Type

	// String returns a string representation of the current value.
	String() string

	// Copy returns a copy of the value, which can be safely modified.
	Copy() Value
}

// NewValue returns new Value of the given Type. The initial value will not
// necessarily be the zero for the type.
func NewValue(typ Type) Value {
	newValue, ok := valueGenerators[typ]
	if !ok {
		return nil
	}
	return newValue()
}

type valueGenerator func() Value

var valueGenerators = map[string]valueGenerator{
	TypeString:          newValueString,
	TypeBinaryString:    newValueBinaryString,
	TypeProtectedString: newValueProtectedString,
	TypeContent:         newValueContent,
	TypeBool:            newValueBool,
	TypeInt:             newValueInt,
	TypeFloat:           newValueFloat,
	TypeDouble:          newValueDouble,
	TypeUDim:            newValueUDim,
	TypeUDim2:           newValueUDim2,
	TypeRay:             newValueRay,
	TypeFaces:           newValueFaces,
	TypeAxes:            newValueAxes,
	TypeBrickColor:      newValueBrickColor,
	TypeColor3:          newValueColor3,
	TypeVector2:         newValueVector2,
	TypeVector3:         newValueVector3,
	TypeCFrame:          newValueCFrame,
	TypeToken:           newValueToken,
	TypeReference:       newValueReference,
	TypeVector3int16:    newValueVector3int16,
	TypeVector2int16:    newValueVector2int16,
	TypeRegion3:         newValueRegion3,
	TypeRegion3int16:    newValueRegion3int16,
}

func joinstr(a ...string) string {
	if len(a) == 0 {
		return ""
	}
	if len(a) == 1 {
		return a[0]
	}
	n := 0
	for i := 0; i < len(a); i++ {
		n += len(a[i])
	}

	b := make([]byte, n)
	bp := 0
	for _, s := range a {
		bp += copy(b[bp:], s)
	}
	return string(b)
}

////////////////////////////////////////////////////////////////
// Values

type ValueString []byte

func newValueString() Value {
	return *new(ValueString)
}

func (ValueString) Type() Type {
	return Typestring
}
func (t ValueString) String() string {
	return string(t)
}
func (t ValueString) Copy() Value {
	c := make(ValueString, len(t))
	copy(c, t)
	return c
}

////////////////

type ValueBinaryString []byte

func newValueBinaryString() Value {
	return *new(ValueBinaryString)
}

func (ValueBinaryString) Type() Type {
	return TypeBinaryString
}
func (t ValueBinaryString) String() string {
	return string(t)
}
func (t ValueBinaryString) Copy() Value {
	c := make(ValueBinaryString, len(t))
	copy(c, t)
	return c
}

////////////////

type ValueProtectedString []byte

func newValueProtectedString() Value {
	return *new(ValueProtectedString)
}

func (ValueProtectedString) Type() Type {
	return TypeProtectedString
}
func (t ValueProtectedString) String() string {
	return string(t)
}
func (t ValueProtectedString) Copy() Value {
	c := make(ValueProtectedString, len(t))
	copy(c, t)
	return c
}

////////////////

type ValueContent []byte

func newValueContent() Value {
	return *new(ValueContent)
}

func (ValueContent) Type() Type {
	return TypeContent
}
func (t ValueContent) String() string {
	return string(t)
}
func (t ValueContent) Copy() Value {
	c := make(ValueContent, len(t))
	copy(c, t)
	return c
}

////////////////

type ValueBool bool

func newValueBool() Value {
	return *new(ValueBool)
}

func (ValueBool) Type() Type {
	return Typebool
}
func (t ValueBool) String() string {
	if t {
		return "true"
	} else {
		return "false"
	}
}
func (t ValueBool) Copy() Value {
	return t
}

////////////////

type ValueInt int32

func newValueInt() Value {
	return *new(ValueInt)
}

func (ValueInt) Type() Type {
	return Typeint
}
func (t ValueInt) String() string {
	return strconv.FormatInt(int64(t), 10)
}
func (t ValueInt) Copy() Value {
	return t
}

////////////////

type ValueFloat float32

func newValueFloat() Value {
	return *new(ValueFloat)
}

func (ValueFloat) Type() Type {
	return Typefloat
}
func (t ValueFloat) String() string {
	return strconv.FormatFloat(float64(t), 'f', -1, 32)
}
func (t ValueFloat) Copy() Value {
	return t
}

////////////////

type ValueDouble float64

func newValueDouble() Value {
	return *new(ValueDouble)
}

func (ValueDouble) Type() Type {
	return Typedouble
}
func (t ValueDouble) String() string {
	return strconv.FormatFloat(float64(t), 'f', -1, 64)
}
func (t ValueDouble) Copy() Value {
	return t
}

////////////////

type ValueUDim struct {
	Scale  float32
	Offset int32
}

func newValueUDim() Value {
	return *new(ValueUDim)
}

func (ValueUDim) Type() Type {
	return TypeUDim
}
func (t ValueUDim) String() string {
	return joinstr(
		strconv.FormatFloat(float64(t.Scale), 'f', -1, 32),
		", ",
		strconv.FormatInt(int64(t.Offset), 10),
	)
}
func (t ValueUDim) Copy() Value {
	return t
}

////////////////

type ValueUDim2 struct {
	X, Y ValueUDim
}

func newValueUDim2() Value {
	return *new(ValueUDim2)
}

func (ValueUDim2) Type() Type {
	return TypeUDim2
}
func (t ValueUDim2) String() string {
	return joinstr(
		"{",
		t.X.String(),
		"}, {",
		t.Y.String(),
		"}",
	)
}
func (t ValueUDim2) Copy() Value {
	return t
}

////////////////

type ValueRay struct {
	Origin, Direction ValueVector3
}

func newValueRay() Value {
	return *new(ValueRay)
}

func (ValueRay) Type() Type {
	return TypeRay
}
func (t ValueRay) String() string {
	return joinstr(
		"{",
		t.Origin.String(),
		"}, {",
		t.Direction.String(),
		"}",
	)
}
func (t ValueRay) Copy() Value {
	return t
}

////////////////

type ValueFaces struct {
	Right, Top, Back, Left, Bottom, Front bool
}

func newValueFaces() Value {
	return *new(ValueFaces)
}

func (ValueFaces) Type() Type {
	return TypeFaces
}
func (t ValueFaces) String() string {
	s := make([]string, 6)
	if t.Front {
		s = append(s, "Front")
	}
	if t.Bottom {
		s = append(s, "Bottom")
	}
	if t.Left {
		s = append(s, "Left")
	}
	if t.Back {
		s = append(s, "Back")
	}
	if t.Top {
		s = append(s, "Top")
	}
	if t.Right {
		s = append(s, "Right")
	}

	return strings.Join(s, ", ")
}
func (t ValueFaces) Copy() Value {
	return t
}

////////////////

type ValueAxes struct {
	X, Y, Z bool
}

func newValueAxes() Value {
	return *new(ValueAxes)
}

func (ValueAxes) Type() Type {
	return TypeAxes
}
func (t ValueAxes) String() string {
	s := make([]string, 3)
	if t.X {
		s = append(s, "X")
	}
	if t.Y {
		s = append(s, "Y")
	}
	if t.Z {
		s = append(s, "Z")
	}

	return strings.Join(s, ", ")
}
func (t ValueAxes) Copy() Value {
	return t
}

////////////////

type ValueBrickColor uint32

func newValueBrickColor() Value {
	return *new(ValueBrickColor)
}

func (ValueBrickColor) Type() Type {
	return TypeBrickColor
}
func (t ValueBrickColor) String() string {
	return strconv.FormatUint(uint64(t), 10)
}
func (t ValueBrickColor) Copy() Value {
	return t
}

func (bc ValueBrickColor) Name() string {
	name, ok := brickColorNames[bc]
	if !ok {
		return brickColorNames[194]
	}

	return name
}

func (bc ValueBrickColor) Color() ValueColor3 {
	color, ok := brickColorColors[bc]
	if !ok {
		return brickColorColors[194]
	}

	return color
}

func (bc ValueBrickColor) Palette() int {
	for i, n := range brickColorPalette {
		if bc == n {
			return i
		}
	}
	return -1
}

////////////////

type ValueColor3 struct {
	R, G, B float32
}

func newValueColor3() Value {
	return *new(ValueColor3)
}

func (ValueColor3) Type() Type {
	return TypeColor3
}
func (t ValueColor3) String() string {
	return joinstr(
		strconv.FormatFloat(float64(t.R), 'f', -1, 32),
		", ",
		strconv.FormatFloat(float64(t.G), 'f', -1, 32),
		", ",
		strconv.FormatFloat(float64(t.B), 'f', -1, 32),
	)
}
func (t ValueColor3) Copy() Value {
	return t
}

////////////////

type ValueVector2 struct {
	X, Y float32
}

func newValueVector2() Value {
	return *new(ValueVector2)
}

func (ValueVector2) Type() Type {
	return TypeVector2
}
func (t ValueVector2) String() string {
	return joinstr(
		strconv.FormatFloat(float64(t.X), 'f', -1, 32),
		", ",
		strconv.FormatFloat(float64(t.Y), 'f', -1, 32),
	)
}
func (t ValueVector2) Copy() Value {
	return t
}

////////////////

type ValueVector3 struct {
	X, Y, Z float32
}

func newValueVector3() Value {
	return *new(ValueVector3)
}

func (ValueVector3) Type() Type {
	return TypeVector3
}
func (t ValueVector3) String() string {
	return joinstr(
		strconv.FormatFloat(float64(t.X), 'f', -1, 32),
		", ",
		strconv.FormatFloat(float64(t.Y), 'f', -1, 32),
		", ",
		strconv.FormatFloat(float64(t.Z), 'f', -1, 32),
	)
}
func (t ValueVector3) Copy() Value {
	return t
}

////////////////

type ValueCFrame struct {
	Position ValueVector3
	Rotation [9]float32
}

func newValueCFrame() Value {
	return *new(ValueCFrame)
}

func (ValueCFrame) Type() Type {
	return TypeCoordinateFrame
}
func (t ValueCFrame) String() string {
	s := make([]string, 12)
	s[0] = strconv.FormatFloat(float64(t.Position.X), 'f', -1, 32)
	s[1] = strconv.FormatFloat(float64(t.Position.Y), 'f', -1, 32)
	s[2] = strconv.FormatFloat(float64(t.Position.Z), 'f', -1, 32)
	for i, f := range t.Rotation {
		s[i+3] = strconv.FormatFloat(float64(f), 'f', -1, 32)
	}
	return strings.Join(s, ", ")
}
func (t ValueCFrame) Copy() Value {
	return t
}

////////////////

type ValueToken uint32

func newValueToken() Value {
	return *new(ValueToken)
}

func (ValueToken) Type() Type {
	return Typetoken
}
func (t ValueToken) String() string {
	return strconv.FormatInt(int64(t), 10)
}
func (t ValueToken) Copy() Value {
	return t
}

////////////////

type ValueReference struct {
	*Instance
}

func newValueReference() Value {
	return *new(ValueReference)
}

func (ValueReference) Type() Type {
	return TypeRef
}
func (t ValueReference) String() string {
	if t.Instance == nil {
		return "<nil>"
	}
	return t.Name()
}
func (t ValueReference) Copy() Value {
	return t
}

////////////////

type ValueVector3int16 struct {
	X, Y, Z int16
}

func newValueVector3int16() Value {
	return *new(ValueVector3int16)
}

func (ValueVector3int16) Type() Type {
	return TypeVector3int16
}
func (t ValueVector3int16) String() string {
	return joinstr(
		strconv.FormatInt(int64(t.X), 10),
		", ",
		strconv.FormatInt(int64(t.Y), 10),
		", ",
		strconv.FormatInt(int64(t.Z), 10),
	)
}
func (t ValueVector3int16) Copy() Value {
	return t
}

////////////////

type ValueVector2int16 struct {
	X, Y int16
}

func newValueVector2int16() Value {
	return *new(ValueVector2int16)
}

func (ValueVector2int16) Type() Type {
	return TypeVector2int16
}
func (t ValueVector2int16) String() string {
	return joinstr(
		strconv.FormatInt(int64(t.X), 10),
		", ",
		strconv.FormatInt(int64(t.Y), 10),
	)
}
func (t ValueVector2int16) Copy() Value {
	return t
}

////////////////

type ValueRegion3 struct {
	CFrame ValueCFrame
	Size   ValueVector3
}

func newValueRegion3() Value {
	return *new(ValueRegion3)
}

func (ValueRegion3) Type() Type {
	return TypeRegion3
}
func (t ValueRegion3) String() string {
	return joinstr(
		t.CFrame.String(),
		"; ",
		t.Size.String(),
	)
}
func (t ValueRegion3) Copy() Value {
	return t
}

////////////////

type ValueRegion3int16 struct {
	Max, Min ValueVector3int16
}

func newValueRegion3int16() Value {
	return *new(ValueRegion3int16)
}

func (ValueRegion3int16) Type() Type {
	return TypeRegion3int16
}
func (t ValueRegion3int16) String() string {
	return joinstr(
		t.Min.String(),
		"; ",
		t.Max.String(),
	)
}
func (t ValueRegion3int16) Copy() Value {
	return t
}

////////////////
