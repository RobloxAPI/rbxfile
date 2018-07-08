package bin

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Type indicates the type of a Value.
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
	TypeInvalid      Type = 0x0
	TypeString       Type = 0x1
	TypeBool         Type = 0x2
	TypeInt          Type = 0x3
	TypeFloat        Type = 0x4
	TypeDouble       Type = 0x5
	TypeUDim         Type = 0x6
	TypeUDim2        Type = 0x7
	TypeRay          Type = 0x8
	TypeFaces        Type = 0x9
	TypeAxes         Type = 0xA
	TypeBrickColor   Type = 0xB
	TypeColor3       Type = 0xC
	TypeVector2      Type = 0xD
	TypeVector3      Type = 0xE
	TypeVector2int16 Type = 0xF
	TypeCFrame       Type = 0x10
	//TypeCFrameQuat Type = 0x11
	TypeToken              Type = 0x12
	TypeReference          Type = 0x13
	TypeVector3int16       Type = 0x14
	TypeNumberSequence     Type = 0x15
	TypeColorSequence      Type = 0x16
	TypeNumberRange        Type = 0x17
	TypeRect2D             Type = 0x18
	TypePhysicalProperties Type = 0x19
	TypeColor3uint8        Type = 0x1A
	TypeInt64              Type = 0x1B
)

var typeStrings = map[Type]string{
	TypeString:       "String",
	TypeBool:         "Bool",
	TypeInt:          "Int",
	TypeFloat:        "Float",
	TypeDouble:       "Double",
	TypeUDim:         "UDim",
	TypeUDim2:        "UDim2",
	TypeRay:          "Ray",
	TypeFaces:        "Faces",
	TypeAxes:         "Axes",
	TypeBrickColor:   "BrickColor",
	TypeColor3:       "Color3",
	TypeVector2:      "Vector2",
	TypeVector3:      "Vector3",
	TypeVector2int16: "Vector2int16",
	TypeCFrame:       "CFrame",
	//TypeCFrameQuat: "CFrameQuat",
	TypeToken:              "Token",
	TypeReference:          "Reference",
	TypeVector3int16:       "Vector3int16",
	TypeNumberSequence:     "NumberSequence",
	TypeColorSequence:      "ColorSequence",
	TypeNumberRange:        "NumberRange",
	TypeRect2D:             "Rect2D",
	TypePhysicalProperties: "PhysicalProperties",
	TypeColor3uint8:        "Color3uint8",
	TypeInt64:              "Int64",
}

// Value is a property value of a certain Type.
type Value interface {
	// Type returns an identifier indicating the type.
	Type() Type

	// FromBytes receives the value of the type from a byte array.
	FromBytes([]byte) error

	// Bytes returns the encoded value of the type as a byte array.
	Bytes() []byte

	// FromArrayBytes receives an array of values of the type from a byte
	// array.
	FromArrayBytes([]byte) ([]Value, error)

	// ArrayBytes returns an array of values of the type encoded as a byte
	// array.
	ArrayBytes([]Value) ([]byte, error)
}

// NewValue returns new Value of the given Type. The initial value will not
// necessarily be the zero for the type. If the given type is invalid, then a
// nil value is returned.
func NewValue(typ Type) Value {
	newValue, ok := valueGenerators[typ]
	if !ok {
		return nil
	}
	return newValue()
}

type valueGenerator func() Value

var valueGenerators = map[Type]valueGenerator{
	TypeString:       newValueString,
	TypeBool:         newValueBool,
	TypeInt:          newValueInt,
	TypeFloat:        newValueFloat,
	TypeDouble:       newValueDouble,
	TypeUDim:         newValueUDim,
	TypeUDim2:        newValueUDim2,
	TypeRay:          newValueRay,
	TypeFaces:        newValueFaces,
	TypeAxes:         newValueAxes,
	TypeBrickColor:   newValueBrickColor,
	TypeColor3:       newValueColor3,
	TypeVector2:      newValueVector2,
	TypeVector3:      newValueVector3,
	TypeVector2int16: newValueVector2int16,
	TypeCFrame:       newValueCFrame,
	//TypeCFrameQuat: newValueCFrameQuat,
	TypeToken:              newValueToken,
	TypeReference:          newValueReference,
	TypeVector3int16:       newValueVector3int16,
	TypeNumberSequence:     newValueNumberSequence,
	TypeColorSequence:      newValueColorSequence,
	TypeNumberRange:        newValueNumberRange,
	TypeRect2D:             newValueRect2D,
	TypePhysicalProperties: newValuePhysicalProperties,
	TypeColor3uint8:        newValueColor3uint8,
	TypeInt64:              newValueInt64,
}

////////////////////////////////////////////////////////////////

// Encodes and decodes a Value based on its fields
type fielder interface {
	// Value.Type
	Type() Type
	// Length of each field
	fieldLen() []int
	// Set bytes of nth field
	fieldSet(int, []byte) error
	// Get bytes of nth field
	fieldGet(int) []byte
}

// Encodes Values that implement the fielder interface.
func interleaveFields(id Type, a []Value) (b []byte, err error) {
	if len(a) == 0 {
		return b, nil
	}

	af := make([]fielder, len(a))
	for i, v := range a {
		af[i] = v.(fielder)
		if af[i].Type() != id {
			return nil, fmt.Errorf("element %d is of type %s where %s is expected", i, af[i].Type().String(), id.String())
		}
	}

	// list is assumed to contain the same kinds of values

	// Number of bytes per field
	nbytes := af[0].fieldLen()
	// Number fields per value
	nfields := len(nbytes)
	// Number of values
	nvalues := len(af)

	// Total bytes per value
	tbytes := 0
	// Offset of each field slice
	ofields := make([]int, len(nbytes)+1)
	for i, n := range nbytes {
		tbytes += n
		ofields[i+1] = ofields[i] + n*nvalues
	}

	b = make([]byte, tbytes*nvalues)

	// List of each field slice
	fields := make([][]byte, nfields)
	for i := range fields {
		// Each field slice affects the final array
		fields[i] = b[ofields[i]:ofields[i+1]]
	}

	for i, v := range af {
		for f, field := range fields {
			fb := v.fieldGet(f)
			if len(fb) != nbytes[f] {
				panic("length of field's bytes does not match given field length")
			}
			copy(field[i*nbytes[f]:], fb)
		}
	}

	// Interleave each field slice independently
	for i, field := range fields {
		if err = interleave(field, nbytes[i]); err != nil {
			return nil, err
		}
	}

	return b, nil
}

// Decodes Values that implement the fielder interface.
func deinterleaveFields(id Type, b []byte) (a []Value, err error) {
	if len(b) == 0 {
		return a, nil
	}

	newValue := valueGenerators[id]
	if newValue == nil {
		return nil, fmt.Errorf("type identifier 0x%X is not a valid Type.", id)
	}

	// Number of bytes per field
	nbytes := newValue().(fielder).fieldLen()
	// Number fields per value
	nfields := len(nbytes)

	// Total bytes per value
	tbytes := 0
	for _, n := range nbytes {
		tbytes += n
	}

	if len(b)%tbytes != 0 {
		return nil, fmt.Errorf("length of array (%d) is not divisible by value byte size (%d)", len(b), tbytes)
	}

	// Number of values
	nvalues := len(b) / tbytes
	// Offset of each field slice
	ofields := make([]int, len(nbytes)+1)
	for i, n := range nbytes {
		ofields[i+1] = ofields[i] + n*nvalues
	}

	a = make([]Value, nvalues)

	// List of each field slice
	fields := make([][]byte, nfields)
	for i := range fields {
		fields[i] = b[ofields[i]:ofields[i+1]]
	}

	// Deinterleave each field slice independently
	for i, field := range fields {
		if err = deinterleave(field, nbytes[i]); err != nil {
			return nil, err
		}
	}

	for i := range a {
		v := newValue()
		vf := v.(fielder)
		for f, field := range fields {
			n := nbytes[f]
			fb := field[i*n : i*n+n]
			vf.fieldSet(f, fb)
		}
		a[i] = v
	}

	return a, nil
}

// Interleave transforms an array of bytes by interleaving them based on a
// given size. The size must be a divisor of the array length.
//
// The array is divided into groups, each `length` in size. The nth elements
// of each group are then moved so that they are group together. For example:
//
//     Original:    abcd1234
//     Interleaved: a1b2c3d4
func interleave(bytes []byte, length int) error {
	if length <= 0 {
		return errors.New("length must be greater than 0")
	}
	if len(bytes)%length != 0 {
		return errors.New("length must be a divisor of array length")
	}

	// Matrix transpose algorithm
	cols := length
	rows := len(bytes) / length
	if rows == cols {
		for r := 0; r < rows; r++ {
			for c := 0; c < r; c++ {
				bytes[r*cols+c], bytes[c*cols+r] = bytes[c*cols+r], bytes[r*cols+c]
			}
		}
	} else {
		tmp := make([]byte, len(bytes))
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				tmp[c*rows+r] = bytes[r*cols+c]
			}
		}
		for i, b := range tmp {
			bytes[i] = b
		}
	}

	return nil
}

func deinterleave(bytes []byte, size int) error {
	if size <= 0 {
		return errors.New("size must be greater than 0")
	}
	if len(bytes)%size != 0 {
		return errors.New("size must be a divisor of array length")
	}

	return interleave(bytes, len(bytes)/size)
}

// Encodes signed integers so that the bytes of negative numbers are more
// similar to positive numbers, making them more compressible.
//
// https://developers.google.com/protocol-buffers/docs/encoding#types
func encodeZigzag32(n int32) uint32 {
	return uint32((n << 1) ^ (n >> 31))
}

func decodeZigzag32(n uint32) int32 {
	return int32((n >> 1) ^ uint32((int32(n&1)<<31)>>31))
}

func encodeZigzag64(n int64) uint64 {
	return uint64((n << 1) ^ (n >> 63))
}

func decodeZigzag64(n uint64) int64 {
	return int64((n >> 1) ^ uint64((int64(n&1)<<63)>>63))
}

// Encodes a Binary32 float with sign at LSB instead of MSB.
func encodeRobloxFloat(f float32) uint32 {
	n := math.Float32bits(f)
	return (n << 1) | (n >> 31)
}

func decodeRobloxFloat(n uint32) float32 {
	f := (n >> 1) | (n << 31)
	return math.Float32frombits(f)
}

// Appends the bytes of a list of Values into a byte array.
func appendValueBytes(id Type, a []Value) (b []byte, err error) {
	for i, v := range a {
		if v.Type() != id {
			return nil, fmt.Errorf("element %d is of type `%s` where `%s` is expected", i, v.Type().String(), id.String())
		}

		b = append(b, v.Bytes()...)
	}

	return b, nil
}

// Reads a byte array as an array of Values of a certain type. Size is the
// byte size of each Value. If size is less than 0, then values are assumed to
// be of variable length. The first 4 bytes of a value is read as length N of
// the value. Field then indicates the size of each field in the value, so the
// next N*field bytes are read as the full value.
func appendByteValues(id Type, b []byte, size int, field int) (a []Value, err error) {
	gen := valueGenerators[id]
	if size < 0 {
		// Variable length; get size from first 4 bytes.
		ba := b
		for len(ba) > 0 {
			if len(ba) < 4 {
				return nil, errors.New("expected 4 more bytes in array")
			}
			size := int(binary.LittleEndian.Uint32(ba))
			if len(ba[4:]) < size*field {
				return nil, fmt.Errorf("expected %d more bytes in array", size*field)
			}

			v := gen()
			if err := v.FromBytes(ba[:4+size*field]); err != nil {
				return nil, err
			}
			a = append(a, v)

			ba = ba[4+size*field:]
		}
	} else {
		for i := 0; i+size <= len(b); i += size {
			v := gen()
			if err := v.FromBytes(b[i : i+size]); err != nil {
				return nil, err
			}
			a = append(a, v)
		}
	}
	return a, nil
}

////////////////////////////////////////////////////////////////

type ValueString []byte

func newValueString() Value {
	return new(ValueString)
}

func (ValueString) Type() Type {
	return TypeString
}

func (v *ValueString) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(v.Type(), a)
}

func (v ValueString) FromArrayBytes(b []byte) (a []Value, err error) {
	return appendByteValues(v.Type(), b, -1, 1)
}

func (v ValueString) Bytes() []byte {
	b := make([]byte, len(v)+4)
	binary.LittleEndian.PutUint32(b, uint32(len(v)))
	copy(b[4:], v)
	return b
}

func (v *ValueString) FromBytes(b []byte) error {
	if len(b) < 4 {
		return errors.New("array length must be greater than or equal to 4")
	}

	length := binary.LittleEndian.Uint32(b[:4])
	str := b[4:]
	if uint32(len(str)) != length {
		return fmt.Errorf("string length (%d) does not match integer length (%d)", len(str), length)
	}

	*v = make(ValueString, len(str))
	copy(*v, str)

	return nil
}

////////////////////////////////////////////////////////////////

type ValueBool bool

func newValueBool() Value {
	return new(ValueBool)
}

func (ValueBool) Type() Type {
	return TypeBool
}

func (v *ValueBool) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(v.Type(), a)
}

func (v ValueBool) FromArrayBytes(b []byte) (a []Value, err error) {
	return appendByteValues(v.Type(), b, 1, 0)
}

func (v ValueBool) Bytes() []byte {
	if v {
		return []byte{1}
	}
	return []byte{0}
}

func (v *ValueBool) FromBytes(b []byte) error {
	if len(b) != 1 {
		return errors.New("array length must be 1")
	}

	*v = b[0] != 0

	return nil
}

////////////////////////////////////////////////////////////////

type ValueInt int32

func newValueInt() Value {
	return new(ValueInt)
}

func (ValueInt) Type() Type {
	return TypeInt
}

func (v *ValueInt) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(v.Type(), a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (v ValueInt) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	if err = deinterleave(bc, 4); err != nil {
		return nil, err
	}

	a, err = appendByteValues(v.Type(), bc, 4, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (v ValueInt) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, encodeZigzag32(int32(v)))
	return b
}

func (v *ValueInt) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	*v = ValueInt(decodeZigzag32(binary.BigEndian.Uint32(b)))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueFloat float32

func newValueFloat() Value {
	return new(ValueFloat)
}

func (ValueFloat) Type() Type {
	return TypeFloat
}

func (v *ValueFloat) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(v.Type(), a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (v ValueFloat) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	if err = deinterleave(bc, 4); err != nil {
		return nil, err
	}

	a, err = appendByteValues(v.Type(), bc, 4, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (v ValueFloat) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, encodeRobloxFloat(float32(v)))
	return b
}

func (v *ValueFloat) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	*v = ValueFloat(decodeRobloxFloat(binary.BigEndian.Uint32(b)))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueDouble float64

func newValueDouble() Value {
	return new(ValueDouble)
}

func (ValueDouble) Type() Type {
	return TypeDouble
}

func (v *ValueDouble) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(v.Type(), a)
}

func (v ValueDouble) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(v.Type(), b, 8, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (v ValueDouble) Bytes() []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, math.Float64bits(float64(v)))
	return b
}

func (v *ValueDouble) FromBytes(b []byte) error {
	if len(b) != 8 {
		return errors.New("array length must be 8")
	}

	*v = ValueDouble(math.Float64frombits(binary.LittleEndian.Uint64(b)))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueUDim struct {
	Scale  ValueFloat
	Offset ValueInt
}

func newValueUDim() Value {
	return new(ValueUDim)
}

func (ValueUDim) Type() Type {
	return TypeUDim
}

func (v *ValueUDim) ArrayBytes(a []Value) (b []byte, err error) {
	return nil, errors.New("not implemented")
}

func (v ValueUDim) FromArrayBytes(b []byte) (a []Value, err error) {
	return nil, errors.New("not implemented")
}

func (v ValueUDim) Bytes() []byte {
	b := make([]byte, 8)

	copy(b[0:4], v.Scale.Bytes())
	copy(b[4:8], v.Offset.Bytes())

	return b
}

func (v *ValueUDim) FromBytes(b []byte) error {
	if len(b) != 8 {
		return errors.New("array length must be 8")
	}

	v.Scale.FromBytes(b[0:4])
	v.Offset.FromBytes(b[4:8])

	return nil
}

////////////////////////////////////////////////////////////////

type ValueUDim2 struct {
	ScaleX  ValueFloat
	ScaleY  ValueFloat
	OffsetX ValueInt
	OffsetY ValueInt
}

func newValueUDim2() Value {
	return new(ValueUDim2)
}

func (ValueUDim2) Type() Type {
	return TypeUDim2
}

func (v ValueUDim2) ArrayBytes(a []Value) (b []byte, err error) {
	return interleaveFields(v.Type(), a)
}

func (v ValueUDim2) FromArrayBytes(b []byte) (a []Value, err error) {
	return deinterleaveFields(v.Type(), b)
}

func (v ValueUDim2) Bytes() []byte {
	b := make([]byte, 16)
	copy(b[0:4], v.ScaleX.Bytes())
	copy(b[4:8], v.ScaleY.Bytes())
	copy(b[8:12], v.OffsetX.Bytes())
	copy(b[12:16], v.OffsetY.Bytes())
	return b
}

func (v *ValueUDim2) FromBytes(b []byte) error {
	if len(b) != 16 {
		return errors.New("array length must be 16")
	}

	v.ScaleX.FromBytes(b[0:4])
	v.ScaleY.FromBytes(b[4:8])
	v.OffsetX.FromBytes(b[8:12])
	v.OffsetY.FromBytes(b[12:16])

	return nil
}

func (ValueUDim2) fieldLen() []int {
	return []int{4, 4, 4, 4}
}

func (v *ValueUDim2) fieldSet(i int, b []byte) (err error) {
	switch i {
	case 0:
		err = v.ScaleX.FromBytes(b)
	case 1:
		err = v.ScaleY.FromBytes(b)
	case 2:
		err = v.OffsetX.FromBytes(b)
	case 3:
		err = v.OffsetY.FromBytes(b)
	}
	return
}

func (v ValueUDim2) fieldGet(i int) (b []byte) {
	switch i {
	case 0:
		return v.ScaleX.Bytes()
	case 1:
		return v.ScaleY.Bytes()
	case 2:
		return v.OffsetX.Bytes()
	case 3:
		return v.OffsetY.Bytes()
	}
	return
}

////////////////////////////////////////////////////////////////

type ValueRay struct {
	OriginX    float32
	OriginY    float32
	OriginZ    float32
	DirectionX float32
	DirectionY float32
	DirectionZ float32
}

func newValueRay() Value {
	return new(ValueRay)
}

func (ValueRay) Type() Type {
	return TypeRay
}

func (v *ValueRay) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(v.Type(), a)
}

func (v ValueRay) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(v.Type(), b, 24, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (v ValueRay) Bytes() []byte {
	b := make([]byte, 24)
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(v.OriginX))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(v.OriginY))
	binary.LittleEndian.PutUint32(b[8:12], math.Float32bits(v.OriginZ))
	binary.LittleEndian.PutUint32(b[12:16], math.Float32bits(v.DirectionX))
	binary.LittleEndian.PutUint32(b[16:20], math.Float32bits(v.DirectionY))
	binary.LittleEndian.PutUint32(b[20:24], math.Float32bits(v.DirectionZ))
	return b
}

func (v *ValueRay) FromBytes(b []byte) error {
	if len(b) != 24 {
		return errors.New("array length must be 24")
	}

	v.OriginX = math.Float32frombits(binary.LittleEndian.Uint32(b[0:4]))
	v.OriginY = math.Float32frombits(binary.LittleEndian.Uint32(b[4:8]))
	v.OriginZ = math.Float32frombits(binary.LittleEndian.Uint32(b[8:12]))
	v.DirectionX = math.Float32frombits(binary.LittleEndian.Uint32(b[12:16]))
	v.DirectionY = math.Float32frombits(binary.LittleEndian.Uint32(b[16:20]))
	v.DirectionZ = math.Float32frombits(binary.LittleEndian.Uint32(b[20:24]))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueFaces struct {
	Right, Top, Back, Left, Bottom, Front bool
}

func newValueFaces() Value {
	return new(ValueFaces)
}

func (ValueFaces) Type() Type {
	return TypeFaces
}

func (v *ValueFaces) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(v.Type(), a)
}

func (v ValueFaces) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(v.Type(), b, 1, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (v ValueFaces) Bytes() []byte {
	flags := [6]bool{v.Right, v.Top, v.Back, v.Left, v.Bottom, v.Front}
	var b byte
	for i, flag := range flags {
		if flag {
			b = b | (1 << uint(i))
		}
	}

	return []byte{b}
}

func (v *ValueFaces) FromBytes(b []byte) error {
	if len(b) != 1 {
		return errors.New("array length must be 1")
	}

	v.Right = b[0]&(1<<0) != 0
	v.Top = b[0]&(1<<1) != 0
	v.Back = b[0]&(1<<2) != 0
	v.Left = b[0]&(1<<3) != 0
	v.Bottom = b[0]&(1<<4) != 0
	v.Front = b[0]&(1<<5) != 0

	return nil
}

////////////////////////////////////////////////////////////////

type ValueAxes struct {
	X, Y, Z bool
}

func newValueAxes() Value {
	return new(ValueAxes)
}

func (ValueAxes) Type() Type {
	return TypeAxes
}

func (v *ValueAxes) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(v.Type(), a)
}

func (v ValueAxes) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(v.Type(), b, 1, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (v ValueAxes) Bytes() []byte {
	flags := [3]bool{v.X, v.Y, v.Z}
	var b byte
	for i, flag := range flags {
		if flag {
			b = b | (1 << uint(i))
		}
	}

	return []byte{b}
}

func (v *ValueAxes) FromBytes(b []byte) error {
	if len(b) != 1 {
		return errors.New("array length must be 1")
	}

	v.X = b[0]&(1<<0) != 0
	v.Y = b[0]&(1<<1) != 0
	v.Z = b[0]&(1<<2) != 0

	return nil
}

////////////////////////////////////////////////////////////////

type ValueBrickColor uint32

func newValueBrickColor() Value {
	return new(ValueBrickColor)
}

func (ValueBrickColor) Type() Type {
	return TypeBrickColor
}

func (v *ValueBrickColor) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(v.Type(), a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (v ValueBrickColor) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	if err = deinterleave(bc, 4); err != nil {
		return nil, err
	}

	a, err = appendByteValues(v.Type(), bc, 4, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (v ValueBrickColor) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v *ValueBrickColor) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	*v = ValueBrickColor(binary.BigEndian.Uint32(b))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueColor3 struct {
	R, G, B ValueFloat
}

func newValueColor3() Value {
	return new(ValueColor3)
}

func (ValueColor3) Type() Type {
	return TypeColor3
}

func (v ValueColor3) ArrayBytes(a []Value) (b []byte, err error) {
	return interleaveFields(v.Type(), a)
}

func (v ValueColor3) FromArrayBytes(b []byte) (a []Value, err error) {
	return deinterleaveFields(v.Type(), b)
}

func (v ValueColor3) Bytes() []byte {
	b := make([]byte, 12)
	copy(b[0:4], v.R.Bytes())
	copy(b[4:8], v.G.Bytes())
	copy(b[8:12], v.B.Bytes())
	return b
}

func (v *ValueColor3) FromBytes(b []byte) error {
	if len(b) != 12 {
		return errors.New("array length must be 12")
	}

	v.R.FromBytes(b[0:4])
	v.G.FromBytes(b[4:8])
	v.B.FromBytes(b[8:12])

	return nil
}

func (ValueColor3) fieldLen() []int {
	return []int{4, 4, 4}
}

func (v *ValueColor3) fieldSet(i int, b []byte) (err error) {
	switch i {
	case 0:
		err = v.R.FromBytes(b)
	case 1:
		err = v.G.FromBytes(b)
	case 2:
		err = v.B.FromBytes(b)
	}
	return
}

func (v ValueColor3) fieldGet(i int) (b []byte) {
	switch i {
	case 0:
		return v.R.Bytes()
	case 1:
		return v.G.Bytes()
	case 2:
		return v.B.Bytes()
	}
	return
}

////////////////////////////////////////////////////////////////

type ValueVector2 struct {
	X, Y ValueFloat
}

func newValueVector2() Value {
	return new(ValueVector2)
}

func (ValueVector2) Type() Type {
	return TypeVector2
}

func (v ValueVector2) ArrayBytes(a []Value) (b []byte, err error) {
	return interleaveFields(v.Type(), a)
}

func (v ValueVector2) FromArrayBytes(b []byte) (a []Value, err error) {
	return deinterleaveFields(v.Type(), b)
}

func (v ValueVector2) Bytes() []byte {
	b := make([]byte, 8)
	copy(b[0:4], v.X.Bytes())
	copy(b[4:8], v.Y.Bytes())
	return b
}

func (v *ValueVector2) FromBytes(b []byte) error {
	if len(b) != 8 {
		return errors.New("array length must be 8")
	}

	v.X.FromBytes(b[0:4])
	v.Y.FromBytes(b[4:8])

	return nil
}

func (ValueVector2) fieldLen() []int {
	return []int{4, 4}
}

func (v *ValueVector2) fieldSet(i int, b []byte) (err error) {
	switch i {
	case 0:
		err = v.X.FromBytes(b)
	case 1:
		err = v.Y.FromBytes(b)
	}
	return
}

func (v ValueVector2) fieldGet(i int) (b []byte) {
	switch i {
	case 0:
		return v.X.Bytes()
	case 1:
		return v.Y.Bytes()
	}
	return
}

////////////////////////////////////////////////////////////////

type ValueVector3 struct {
	X, Y, Z ValueFloat
}

func newValueVector3() Value {
	return new(ValueVector3)
}

func (ValueVector3) Type() Type {
	return TypeVector3
}

func (v ValueVector3) ArrayBytes(a []Value) (b []byte, err error) {
	return interleaveFields(v.Type(), a)
}

func (v ValueVector3) FromArrayBytes(b []byte) (a []Value, err error) {
	return deinterleaveFields(v.Type(), b)
}

func (v ValueVector3) Bytes() []byte {
	b := make([]byte, 12)
	copy(b[0:4], v.X.Bytes())
	copy(b[4:8], v.Y.Bytes())
	copy(b[8:12], v.Z.Bytes())
	return b
}

func (v *ValueVector3) FromBytes(b []byte) error {
	if len(b) != 12 {
		return errors.New("array length must be 12")
	}

	v.X.FromBytes(b[0:4])
	v.Y.FromBytes(b[4:8])
	v.Z.FromBytes(b[8:12])

	return nil
}

func (ValueVector3) fieldLen() []int {
	return []int{4, 4, 4}
}

func (v *ValueVector3) fieldSet(i int, b []byte) (err error) {
	switch i {
	case 0:
		err = v.X.FromBytes(b)
	case 1:
		err = v.Y.FromBytes(b)
	case 2:
		err = v.Z.FromBytes(b)
	}
	return
}

func (v ValueVector3) fieldGet(i int) (b []byte) {
	switch i {
	case 0:
		return v.X.Bytes()
	case 1:
		return v.Y.Bytes()
	case 2:
		return v.Z.Bytes()
	}
	return
}

////////////////////////////////////////////////////////////////

type ValueVector2int16 struct {
	X, Y int16
}

func newValueVector2int16() Value {
	return new(ValueVector2int16)
}

func (ValueVector2int16) Type() Type {
	return TypeVector2int16
}

func (v *ValueVector2int16) ArrayBytes(a []Value) (b []byte, err error) {
	return nil, errors.New("not implemented")
}

func (v ValueVector2int16) FromArrayBytes(b []byte) (a []Value, err error) {
	return nil, errors.New("not implemented")
}

func (v ValueVector2int16) Bytes() []byte {
	b := make([]byte, 4)

	binary.LittleEndian.PutUint16(b[0:2], uint16(v.X))
	binary.LittleEndian.PutUint16(b[2:4], uint16(v.Y))

	return b
}

func (v *ValueVector2int16) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	v.X = int16(binary.LittleEndian.Uint16(b[0:2]))
	v.Y = int16(binary.LittleEndian.Uint16(b[2:4]))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueCFrame struct {
	Special  uint8
	Rotation [9]float32
	Position ValueVector3
}

func newValueCFrame() Value {
	return new(ValueCFrame)
}

func (ValueCFrame) Type() Type {
	return TypeCFrame
}

func (v *ValueCFrame) ArrayBytes(a []Value) (b []byte, err error) {
	p := make([]Value, len(a))

	for i, cf := range a {
		cf, ok := cf.(*ValueCFrame)
		if !ok {
			return nil, fmt.Errorf("element %d is of type `%s` where `%s` is expected", i, cf.Type().String(), v.Type().String())
		}

		// Build matrix part.
		b = append(b, cf.Special)
		if cf.Special == 0 {
			r := make([]byte, len(cf.Rotation)*4)
			for i, f := range cf.Rotation {
				binary.LittleEndian.PutUint32(r[i*4:i*4+4], math.Float32bits(f))
			}
			b = append(b, r...)
		}

		// Prepare position part.
		p[i] = &cf.Position
	}

	// Build position part.
	pb, _ := v.Position.ArrayBytes(p)
	b = append(b, pb...)

	return b, nil
}

func (v ValueCFrame) FromArrayBytes(b []byte) (a []Value, err error) {
	cfs := make([]*ValueCFrame, 0)

	// This loop reads the matrix data. i is the current position in the byte
	// array. n is the expected size of the position data, which increases
	// every time another CFrame is read. As long as the number of remaining
	// bytes is greater than n, then the next byte can be assumed to be matrix
	// data. By the end, the number of remaining bytes should be exactly equal
	// to n.
	i := 0
	for n := 0; len(b)-i > n; n += 12 {
		cf := new(ValueCFrame)

		cf.Special = b[i]
		i++

		if cf.Special == 0 {
			q := len(cf.Rotation) * 4
			r := b[i:]
			if len(r) < q {
				return nil, fmt.Errorf("expected %d more bytes in array", q)
			}
			for i := range cf.Rotation {
				cf.Rotation[i] = math.Float32frombits(binary.LittleEndian.Uint32(r[i*4 : i*4+4]))
			}
			i += q
		}

		cfs = append(cfs, cf)
	}

	// Read remaining position data using the Position field, which is a
	// ValueVector3. FromArrayBytes doesn't modify the value, so it's safe to
	// use from a non-pointer ValueVector3.
	a, err = v.Position.FromArrayBytes(b[i:])
	if err != nil {
		return nil, err
	}

	if len(a) != len(cfs) {
		return nil, errors.New("number of positions does not match number of matrices")
	}

	// Hack: use 'a' variable to receive Vector3 values, then replace them
	// with CFrames. This lets us avoid needing to copy 'cfs' to 'a', and
	// needing to create a second array.
	for i, p := range a {
		cfs[i].Position = *p.(*ValueVector3)
		a[i] = cfs[i]
	}

	return a, err
}

func (v ValueCFrame) Bytes() []byte {
	var b []byte
	if v.Special == 0 {
		b = make([]byte, 49)
		r := b[1:]
		for i, f := range v.Rotation {
			binary.LittleEndian.PutUint32(r[i*4:i*4+4], math.Float32bits(f))
		}
	} else {
		b = make([]byte, 13)
		b[0] = v.Special
	}

	copy(b[len(b)-12:], v.Position.Bytes())

	return b
}

func (v *ValueCFrame) FromBytes(b []byte) error {
	if b[0] == 0 && len(b) != 49 {
		return errors.New("array length must be 49")
	} else if b[0] != 0 && len(b) != 13 {
		return errors.New("array length must be 13")
	}

	v.Special = b[0]

	if b[0] == 0 {
		r := b[1:]
		for i := range v.Rotation {
			v.Rotation[i] = math.Float32frombits(binary.LittleEndian.Uint32(r[i*4 : i*4+4]))
		}
	} else {
		for i := range v.Rotation {
			v.Rotation[i] = 0
		}
	}

	v.Position.FromBytes(b[len(b)-12:])

	return nil
}

////////////////////////////////////////////////////////////////

type ValueToken uint32

func newValueToken() Value {
	return new(ValueToken)
}

func (ValueToken) Type() Type {
	return TypeToken
}

func (v *ValueToken) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(v.Type(), a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (v ValueToken) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	if err = deinterleave(bc, 4); err != nil {
		return nil, err
	}

	a, err = appendByteValues(v.Type(), bc, 4, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (v ValueToken) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v *ValueToken) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	*v = ValueToken(binary.BigEndian.Uint32(b))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueReference int32

func newValueReference() Value {
	return new(ValueReference)
}

func (ValueReference) Type() Type {
	return TypeReference
}

func (v *ValueReference) ArrayBytes(a []Value) (b []byte, err error) {
	if len(a) == 0 {
		return b, nil
	}

	size := 4
	b = make([]byte, len(a)*size)

	var prev ValueReference
	for i, ref := range a {
		ref, ok := ref.(*ValueReference)
		if !ok {
			return nil, fmt.Errorf("value %d is of type `%s` where `%s` is expected", i, ref.Type().String(), v.Type().String())
		}

		if i == 0 {
			copy(b[i*size:i*size+size], ref.Bytes())
		} else {
			// Convert absolute ref to relative ref.
			copy(b[i*size:i*size+size], (*ref - prev).Bytes())
		}

		prev = *ref
	}

	if err = interleave(b, size); err != nil {
		return nil, err
	}

	return b, nil
}

func (v ValueReference) FromArrayBytes(b []byte) (a []Value, err error) {
	if len(b) == 0 {
		return a, nil
	}

	size := 4
	if len(b)%size != 0 {
		return nil, fmt.Errorf("array must be divisible by %d", size)
	}

	bc := make([]byte, len(b))
	copy(bc, b)
	if err = deinterleave(bc, size); err != nil {
		return nil, err
	}

	a = make([]Value, len(bc)/size)
	for i := 0; i < len(bc)/size; i++ {
		ref := new(ValueReference)
		ref.FromBytes(bc[i*size : i*size+size])

		if i > 0 {
			// Convert relative ref to absolute ref.
			r := *a[i-1].(*ValueReference)
			*ref = r + *ref
		}

		a[i] = ref
	}

	return a, nil
}

func (v ValueReference) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, encodeZigzag32(int32(v)))
	return b
}

func (v *ValueReference) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	*v = ValueReference(decodeZigzag32(binary.BigEndian.Uint32(b)))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueVector3int16 struct {
	X, Y, Z int16
}

func newValueVector3int16() Value {
	return new(ValueVector3int16)
}

func (ValueVector3int16) Type() Type {
	return TypeVector3int16
}

func (v *ValueVector3int16) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(v.Type(), a)
}

func (v ValueVector3int16) FromArrayBytes(b []byte) (a []Value, err error) {
	return appendByteValues(v.Type(), b, 6, 0)
}

func (v ValueVector3int16) Bytes() []byte {
	b := make([]byte, 6)

	binary.LittleEndian.PutUint16(b[0:2], uint16(v.X))
	binary.LittleEndian.PutUint16(b[2:4], uint16(v.Y))
	binary.LittleEndian.PutUint16(b[4:6], uint16(v.Z))

	return b
}

func (v *ValueVector3int16) FromBytes(b []byte) error {
	if len(b) != 6 {
		return errors.New("array length must be 6")
	}

	v.X = int16(binary.LittleEndian.Uint16(b[0:2]))
	v.Y = int16(binary.LittleEndian.Uint16(b[2:4]))
	v.Z = int16(binary.LittleEndian.Uint16(b[4:6]))

	return nil
}

////////////////////////////////////////////////////////////////

const sizeNSK = 3 * 4

type ValueNumberSequenceKeypoint struct {
	Time, Value, Envelope float32
}

type ValueNumberSequence []ValueNumberSequenceKeypoint

func newValueNumberSequence() Value {
	return new(ValueNumberSequence)
}

func (ValueNumberSequence) Type() Type {
	return TypeNumberSequence
}

func (v *ValueNumberSequence) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(v.Type(), a)
}

func (v ValueNumberSequence) FromArrayBytes(b []byte) (a []Value, err error) {
	return appendByteValues(v.Type(), b, -1, sizeNSK)
}

func (v ValueNumberSequence) Bytes() []byte {
	b := make([]byte, 4+len(v)*sizeNSK)

	binary.LittleEndian.PutUint32(b, uint32(len(v)))
	ba := b[4:]

	for i, nsk := range v {
		bk := ba[i*sizeNSK:]
		binary.LittleEndian.PutUint32(bk[0:4], math.Float32bits(nsk.Time))
		binary.LittleEndian.PutUint32(bk[4:8], math.Float32bits(nsk.Value))
		binary.LittleEndian.PutUint32(bk[8:12], math.Float32bits(nsk.Envelope))
	}

	return b
}

func (v *ValueNumberSequence) FromBytes(b []byte) error {
	if len(b) < 4 {
		return errors.New("array length must be at least 4")
	}

	length := int(binary.LittleEndian.Uint32(b))
	ba := b[4:]
	if len(ba) != sizeNSK*length {
		return fmt.Errorf("expected array length of %d (4 + %d * %d)", 4+sizeNSK*length, sizeNSK, length)
	}

	a := make(ValueNumberSequence, length)
	for i := 0; i < length; i++ {
		bk := ba[i*sizeNSK:]
		a[i] = ValueNumberSequenceKeypoint{
			Time:     math.Float32frombits(binary.LittleEndian.Uint32(bk[0:4])),
			Value:    math.Float32frombits(binary.LittleEndian.Uint32(bk[4:8])),
			Envelope: math.Float32frombits(binary.LittleEndian.Uint32(bk[8:12])),
		}
	}

	*v = a

	return nil
}

////////////////////////////////////////////////////////////////

const sizeCSK = 4 + 3*4 + 4

type ValueColorSequenceKeypoint struct {
	Time     float32
	Value    ValueColor3
	Envelope float32
}

type ValueColorSequence []ValueColorSequenceKeypoint

func newValueColorSequence() Value {
	return new(ValueColorSequence)
}

func (ValueColorSequence) Type() Type {
	return TypeColorSequence
}

func (v *ValueColorSequence) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(v.Type(), a)
}

func (v ValueColorSequence) FromArrayBytes(b []byte) (a []Value, err error) {
	return appendByteValues(v.Type(), b, -1, sizeCSK)
}

func (v ValueColorSequence) Bytes() []byte {
	b := make([]byte, 4+len(v)*sizeCSK)

	binary.LittleEndian.PutUint32(b, uint32(len(v)))
	ba := b[4:]

	for i, csk := range v {
		bk := ba[i*sizeCSK:]
		binary.LittleEndian.PutUint32(bk[0:4], math.Float32bits(csk.Time))
		binary.LittleEndian.PutUint32(bk[4:8], math.Float32bits(float32(csk.Value.R)))
		binary.LittleEndian.PutUint32(bk[8:12], math.Float32bits(float32(csk.Value.G)))
		binary.LittleEndian.PutUint32(bk[12:16], math.Float32bits(float32(csk.Value.B)))
		binary.LittleEndian.PutUint32(bk[16:20], math.Float32bits(csk.Envelope))
	}

	return b
}

func (v *ValueColorSequence) FromBytes(b []byte) error {
	if len(b) < 4 {
		return errors.New("array length must be at least 4")
	}

	length := int(binary.LittleEndian.Uint32(b))
	ba := b[4:]
	if len(ba) != sizeCSK*length {
		return fmt.Errorf("expected array length of %d (4 + %d * %d)", 4+sizeCSK*length, sizeCSK, length)
	}

	a := make(ValueColorSequence, length)
	for i := 0; i < length; i++ {
		bk := ba[i*sizeCSK:]
		c3 := *new(ValueColor3)
		c3.FromBytes(bk[4:16])
		a[i] = ValueColorSequenceKeypoint{
			Time: math.Float32frombits(binary.LittleEndian.Uint32(bk[0:4])),
			Value: ValueColor3{
				R: ValueFloat(math.Float32frombits(binary.LittleEndian.Uint32(bk[4:8]))),
				G: ValueFloat(math.Float32frombits(binary.LittleEndian.Uint32(bk[8:12]))),
				B: ValueFloat(math.Float32frombits(binary.LittleEndian.Uint32(bk[12:16]))),
			},
			Envelope: math.Float32frombits(binary.LittleEndian.Uint32(bk[16:20])),
		}
	}

	*v = a

	return nil
}

////////////////////////////////////////////////////////////////

type ValueNumberRange struct {
	Min, Max float32
}

func newValueNumberRange() Value {
	return new(ValueNumberRange)
}

func (ValueNumberRange) Type() Type {
	return TypeNumberRange
}

func (v *ValueNumberRange) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(v.Type(), a)
}

func (v ValueNumberRange) FromArrayBytes(b []byte) (a []Value, err error) {
	return appendByteValues(v.Type(), b, 8, 0)
}

func (v ValueNumberRange) Bytes() []byte {
	b := make([]byte, 8)

	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(v.Min))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(v.Max))

	return b
}

func (v *ValueNumberRange) FromBytes(b []byte) error {
	if len(b) != 8 {
		return errors.New("array length must be 8")
	}

	v.Min = math.Float32frombits(binary.LittleEndian.Uint32(b[0:4]))
	v.Max = math.Float32frombits(binary.LittleEndian.Uint32(b[4:8]))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueRect2D struct {
	Min, Max ValueVector2
}

func newValueRect2D() Value {
	return new(ValueRect2D)
}

func (ValueRect2D) Type() Type {
	return TypeRect2D
}

func (v ValueRect2D) ArrayBytes(a []Value) (b []byte, err error) {
	return interleaveFields(v.Type(), a)
}

func (v ValueRect2D) FromArrayBytes(b []byte) (a []Value, err error) {
	return deinterleaveFields(v.Type(), b)
}

func (v ValueRect2D) Bytes() []byte {
	b := make([]byte, 16)

	copy(b[0:8], v.Min.Bytes())
	copy(b[8:16], v.Max.Bytes())

	return b
}

func (v *ValueRect2D) FromBytes(b []byte) error {
	if len(b) != 16 {
		return errors.New("array length must be 16")
	}

	v.Min.FromBytes(b[0:8])
	v.Max.FromBytes(b[8:16])

	return nil
}

func (ValueRect2D) fieldLen() []int {
	return []int{4, 4, 4, 4}
}

func (v *ValueRect2D) fieldSet(i int, b []byte) (err error) {
	switch i {
	case 0:
		err = v.Min.X.FromBytes(b)
	case 1:
		err = v.Min.Y.FromBytes(b)
	case 2:
		err = v.Max.X.FromBytes(b)
	case 3:
		err = v.Max.Y.FromBytes(b)
	}
	return
}

func (v ValueRect2D) fieldGet(i int) (b []byte) {
	switch i {
	case 0:
		return v.Min.X.Bytes()
	case 1:
		return v.Min.Y.Bytes()
	case 2:
		return v.Max.X.Bytes()
	case 3:
		return v.Max.Y.Bytes()
	}
	return
}

////////////////////////////////////////////////////////////////

type ValuePhysicalProperties struct {
	CustomPhysics    byte
	Density          float32
	Friction         float32
	Elasticity       float32
	FrictionWeight   float32
	ElasticityWeight float32
}

func newValuePhysicalProperties() Value {
	return new(ValuePhysicalProperties)
}

func (ValuePhysicalProperties) Type() Type {
	return TypePhysicalProperties
}

func (v *ValuePhysicalProperties) ArrayBytes(a []Value) (b []byte, err error) {
	q := make([]byte, 20)
	for i, pp := range a {
		pp, ok := pp.(*ValuePhysicalProperties)
		if !ok {
			return nil, fmt.Errorf("element %d is of type `%s` where `%s` is expected", i, pp.Type().String(), v.Type().String())
		}

		b = append(b, v.CustomPhysics)
		if v.CustomPhysics != 0 {
			binary.LittleEndian.PutUint32(q[0*4:0*4+4], math.Float32bits(pp.Density))
			binary.LittleEndian.PutUint32(q[1*4:1*4+4], math.Float32bits(pp.Friction))
			binary.LittleEndian.PutUint32(q[2*4:2*4+4], math.Float32bits(pp.Elasticity))
			binary.LittleEndian.PutUint32(q[3*4:3*4+4], math.Float32bits(pp.FrictionWeight))
			binary.LittleEndian.PutUint32(q[4*4:4*4+4], math.Float32bits(pp.ElasticityWeight))
			b = append(b, q...)
		}
	}
	return b, nil
}

func (v ValuePhysicalProperties) FromArrayBytes(b []byte) (a []Value, err error) {
	for i := 0; i < len(b); {
		pp := new(ValuePhysicalProperties)
		pp.CustomPhysics = b[i]
		i++
		if pp.CustomPhysics != 0 {
			const size = 5 * 4
			p := b[i:]
			if len(p) < size {
				return nil, fmt.Errorf("expected %d more bytes in array", size)
			}
			pp.Density = math.Float32frombits(binary.LittleEndian.Uint32(p[0*4 : 0*4+4]))
			pp.Friction = math.Float32frombits(binary.LittleEndian.Uint32(p[1*4 : 1*4+4]))
			pp.Elasticity = math.Float32frombits(binary.LittleEndian.Uint32(p[2*4 : 2*4+4]))
			pp.FrictionWeight = math.Float32frombits(binary.LittleEndian.Uint32(p[3*4 : 3*4+4]))
			pp.ElasticityWeight = math.Float32frombits(binary.LittleEndian.Uint32(p[4*4 : 4*4+4]))
			i += size
		}
		a = append(a, pp)
	}
	return a, err
}

func (v ValuePhysicalProperties) Bytes() []byte {
	if v.CustomPhysics != 0 {
		b := make([]byte, 21)
		b[0] = v.CustomPhysics
		q := b[1:]
		binary.LittleEndian.PutUint32(q[0*4:0*4+4], math.Float32bits(v.Density))
		binary.LittleEndian.PutUint32(q[1*4:1*4+4], math.Float32bits(v.Friction))
		binary.LittleEndian.PutUint32(q[2*4:2*4+4], math.Float32bits(v.Elasticity))
		binary.LittleEndian.PutUint32(q[3*4:3*4+4], math.Float32bits(v.FrictionWeight))
		binary.LittleEndian.PutUint32(q[4*4:4*4+4], math.Float32bits(v.ElasticityWeight))
		return b
	}
	return make([]byte, 1)
}

func (v *ValuePhysicalProperties) FromBytes(b []byte) error {
	if b[0] == 0 && len(b) != 21 {
		return errors.New("array length must be 21")
	} else if b[0] != 0 && len(b) != 1 {
		return errors.New("array length must be 1")
	}

	v.CustomPhysics = b[0]
	if v.CustomPhysics != 0 {
		p := b[1:]
		v.Density = math.Float32frombits(binary.LittleEndian.Uint32(p[0*4 : 0*4+4]))
		v.Friction = math.Float32frombits(binary.LittleEndian.Uint32(p[1*4 : 1*4+4]))
		v.Elasticity = math.Float32frombits(binary.LittleEndian.Uint32(p[2*4 : 2*4+4]))
		v.FrictionWeight = math.Float32frombits(binary.LittleEndian.Uint32(p[3*4 : 3*4+4]))
		v.ElasticityWeight = math.Float32frombits(binary.LittleEndian.Uint32(p[4*4 : 4*4+4]))
	} else {
		v.Density = 0
		v.Friction = 0
		v.Elasticity = 0
		v.FrictionWeight = 0
		v.ElasticityWeight = 0
	}

	return nil
}

////////////////////////////////////////////////////////////////

type ValueColor3uint8 struct {
	R, G, B byte
}

func newValueColor3uint8() Value {
	return new(ValueColor3uint8)
}

func (ValueColor3uint8) Type() Type {
	return TypeColor3uint8
}

func (v ValueColor3uint8) ArrayBytes(a []Value) (b []byte, err error) {
	return interleaveFields(v.Type(), a)
}

func (v ValueColor3uint8) FromArrayBytes(b []byte) (a []Value, err error) {
	return deinterleaveFields(v.Type(), b)
}

func (v ValueColor3uint8) Bytes() []byte {
	b := make([]byte, 3)
	b[0] = v.R
	b[1] = v.G
	b[2] = v.B
	return b
}

func (v *ValueColor3uint8) FromBytes(b []byte) error {
	if len(b) != 3 {
		return errors.New("array length must be 3")
	}

	v.R = b[0]
	v.G = b[1]
	v.B = b[2]

	return nil
}

func (ValueColor3uint8) fieldLen() []int {
	return []int{1, 1, 1}
}

func (v *ValueColor3uint8) fieldSet(i int, b []byte) (err error) {
	switch i {
	case 0:
		v.R = b[0]
	case 1:
		v.G = b[0]
	case 2:
		v.B = b[0]
	}
	return
}

func (v ValueColor3uint8) fieldGet(i int) (b []byte) {
	switch i {
	case 0:
		return []byte{v.R}
	case 1:
		return []byte{v.G}
	case 2:
		return []byte{v.B}
	}
	return
}

////////////////////////////////////////////////////////////////

type ValueInt64 int64

func newValueInt64() Value {
	return new(ValueInt64)
}

func (ValueInt64) Type() Type {
	return TypeInt64
}

func (v *ValueInt64) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(v.Type(), a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 8); err != nil {
		return nil, err
	}

	return b, nil
}

func (v ValueInt64) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	if err = deinterleave(bc, 8); err != nil {
		return nil, err
	}

	a, err = appendByteValues(v.Type(), bc, 8, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (v ValueInt64) Bytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, encodeZigzag64(int64(v)))
	return b
}

func (v *ValueInt64) FromBytes(b []byte) error {
	if len(b) != 8 {
		return errors.New("array length must be 8")
	}

	*v = ValueInt64(decodeZigzag64(binary.BigEndian.Uint64(b)))

	return nil
}

////////////////////////////////////////////////////////////////
