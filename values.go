package rbxfile

import (
	"strconv"
	"strings"
)

// Value holds a value of a particular Roblox type.
type Value interface {
	// TypeString returns the name of the type.
	TypeString() string

	// String returns a string representation of the current value.
	String() string

	// Copy returns a copy of the value.
	Copy() Value
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

func (ValueString) TypeString() string {
	return "string"
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

func (ValueBinaryString) TypeString() string {
	return "BinaryString"
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

func (ValueProtectedString) TypeString() string {
	return "ProtectedString"
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

func (ValueContent) TypeString() string {
	return "Content"
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

func (ValueBool) TypeString() string {
	return "bool"
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

func (ValueInt) TypeString() string {
	return "int"
}
func (t ValueInt) String() string {
	return strconv.FormatInt(int64(t), 10)
}
func (t ValueInt) Copy() Value {
	return t
}

////////////////

type ValueFloat float32

func (ValueFloat) TypeString() string {
	return "float"
}
func (t ValueFloat) String() string {
	return strconv.FormatFloat(float64(t), 'f', -1, 32)
}
func (t ValueFloat) Copy() Value {
	return t
}

////////////////

type ValueDouble float64

func (ValueDouble) TypeString() string {
	return "double"
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

func (ValueUDim) TypeString() string {
	return "UDim"
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

func (ValueUDim2) TypeString() string {
	return "UDim2"
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

func (ValueRay) TypeString() string {
	return "Ray"
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

func (ValueFaces) TypeString() string {
	return "Faces"
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

func (ValueAxes) TypeString() string {
	return "Axes"
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

func (ValueBrickColor) TypeString() string {
	return "BrickColor"
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

func (ValueColor3) TypeString() string {
	return "Color3"
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

func (ValueVector2) TypeString() string {
	return "Vector2"
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

func (ValueVector3) TypeString() string {
	return "Vector3"
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
	X, Y, Z float32
	R       [9]float32
}

func (ValueCFrame) TypeString() string {
	return "CoordinateFrame"
}
func (t ValueCFrame) String() string {
	s := make([]string, 12)
	s[0] = strconv.FormatFloat(float64(t.X), 'f', -1, 32)
	s[1] = strconv.FormatFloat(float64(t.Y), 'f', -1, 32)
	s[2] = strconv.FormatFloat(float64(t.Z), 'f', -1, 32)
	for i, f := range t.R {
		s[i+3] = strconv.FormatFloat(float64(f), 'f', -1, 32)
	}
	return strings.Join(s, ", ")
}
func (t ValueCFrame) Copy() Value {
	return t
}

////////////////

type ValueToken int32

func (ValueToken) TypeString() string {
	return "token"
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

func (ValueReference) TypeString() string {
	return "Ref"
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

func (ValueVector3int16) TypeString() string {
	return "Vector3int16"
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

func (ValueVector2int16) TypeString() string {
	return "Vector2int16"
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

func (ValueRegion3) TypeString() string {
	return "Region3"
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

func (ValueRegion3int16) TypeString() string {
	return "Region3int16"
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
