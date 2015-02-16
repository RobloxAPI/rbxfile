package bin

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Value is a property value of a certain type.
type Value interface {
	// TypeID returns an identifier indicating the type.
	TypeID() byte

	// TypeString returns the name of the type.
	TypeString() string

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

type valueGenerator func() Value

var valueGenerators = map[byte]valueGenerator{
	newValueString().TypeID():       newValueString,
	newValueBool().TypeID():         newValueBool,
	newValueInt().TypeID():          newValueInt,
	newValueFloat().TypeID():        newValueFloat,
	newValueDouble().TypeID():       newValueDouble,
	newValueUDim().TypeID():         newValueUDim,
	newValueUDim2().TypeID():        newValueUDim2,
	newValueRay().TypeID():          newValueRay,
	newValueFaces().TypeID():        newValueFaces,
	newValueAxes().TypeID():         newValueAxes,
	newValueBrickColor().TypeID():   newValueBrickColor,
	newValueColor3().TypeID():       newValueColor3,
	newValueVector2().TypeID():      newValueVector2,
	newValueVector3().TypeID():      newValueVector3,
	newValueVector2int16().TypeID(): newValueVector2int16,
	newValueCFrame().TypeID():       newValueCFrame,
	//0x11: newValueCFrameQuat,
	newValueToken().TypeID():        newValueToken,
	newValueReference().TypeID():    newValueReference,
	newValueVector3int16().TypeID(): newValueVector3int16,
}

////////////////////////////////////////////////////////////////

// Interleave transforms an array of bytes by interleaving them based on a
// given size. The size must be a divisor of the array length.
//
// The array is divided into groups, each `length` in size. The nth elements
// of each group are then moved so that they are group together. For example:
//
//     Original:    abcd1234
//     Interleaved: a1b2c3d4
//
// Same as bigInterleave with a size of 1.
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

// bigInterleave transforms an array of bytes by interleaving them based on a
// given size. The size must be a divisor of the array length. Bytes in the
// array are grouped into single units, each `size` in length.
//
// These units are then grouped together into sections, each section being
// `length` units in size (or length*size bytes). The nth units of each
// section are then moved so that they are grouped together. e.g.
//     Original:    abcd1234
//     Interleaved: a1b2c3d4
func bigInterleave(bytes []byte, size, length int) error {
	if size <= 0 {
		return errors.New("size must be greater than 0")
	}
	if len(bytes)%size != 0 {
		return errors.New("size must be a divisor of array length")
	}

	arrayLen := len(bytes) / size

	if length <= 0 {
		return errors.New("length must be greater than 0")
	}
	if arrayLen%length != 0 {
		return errors.New("length must be a divisor of grouped array length")
	}

	// Matrix transpose algorithm
	tmp := make([]byte, len(bytes))

	cols := length
	rows := arrayLen / cols

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			si := (c*rows + r) * size
			gi := (r*cols + c) * size
			copy(tmp[si:si+size], bytes[gi:gi+size])
		}
	}

	for i, b := range tmp {
		bytes[i] = b
	}

	return nil
}

func bigDeinterleave(bytes []byte, size, length int) error {
	if size <= 0 {
		return errors.New("size must be greater than 0")
	}
	if len(bytes)%size != 0 {
		return errors.New("size must be a divisor of array length")
	}

	arrayLen := len(bytes) / size

	if length <= 0 {
		return errors.New("length must be greater than 0")
	}
	if arrayLen%length != 0 {
		return errors.New("length must be a divisor of grouped array length")
	}

	return bigInterleave(bytes, size, arrayLen/length)
}

// Encodes signed integers so that the bytes of negative numbers are more
// similar to positive numbers, making them more compressible.
//
// https://developers.google.com/protocol-buffers/docs/encoding#types
func encodeZigzag(n int32) uint32 {
	return uint32((n << 1) ^ (n >> 31))
}

func decodeZigzag(n uint32) int32 {
	return int32((n >> 1) ^ uint32((int32(n&1)<<31)>>31))
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
func appendValueBytes(t Value, a []Value) (b []byte, err error) {
	for i, v := range a {
		if v.TypeID() != t.TypeID() {
			return nil, errors.New(
				fmt.Sprintf("element %d is of type `%s` where `%s` is expected", i, v.TypeString(), t.TypeString()),
			)
		}

		b = append(b, v.Bytes()...)
	}

	return b, nil
}

// Reads a byte array as an array of Values of a certain type. Size is the
// byte size of each Value.
func appendByteValues(id byte, b []byte, size int) (a []Value, err error) {
	gen := valueGenerators[id]
	for i := 0; i+size <= len(b); i += size {
		v := gen()
		if err := v.FromBytes(b[i : i+size]); err != nil {
			return nil, err
		}
		a = append(a, v)
	}
	return a, nil
}

////////////////////////////////////////////////////////////////

type ValueString []byte

func newValueString() Value {
	return new(ValueString)
}

func (ValueString) TypeID() byte {
	return 0x1
}

func (ValueString) TypeString() string {
	return "String"
}

func (t *ValueString) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t ValueString) FromArrayBytes(b []byte) (a []Value, err error) {
	for i := 0; i < len(b); {
		if i+4 > len(b) {
			return nil, errors.New("array element must be at least 4 bytes")
		}
		length := binary.LittleEndian.Uint32(b[i:])

		if i+4+int(length) > len(b) {
			return nil, errors.New(fmt.Sprintf("array length (%d) must be at least 4+%d bytes", len(b), length))
		}

		v := new(ValueString)
		if err = v.FromBytes(b[i : i+4+int(length)]); err != nil {
			return nil, err
		}
		a = append(a, v)

		i += 4 + int(length)
	}

	return a, nil
}

func (t ValueString) Bytes() []byte {
	b := make([]byte, len(t)+4)
	binary.LittleEndian.PutUint32(b, uint32(len(t)))
	copy(b[4:], t)
	return b
}

func (t *ValueString) FromBytes(b []byte) error {
	if len(b) < 4 {
		return errors.New("array length must be greater than or equal to 4")
	}

	length := binary.LittleEndian.Uint32(b[:4])
	str := b[4:]
	if uint32(len(str)) != length {
		return errors.New(fmt.Sprintf("string length (%d) does not match integer length (%d)", len(str), length))
	}

	*t = make(ValueString, len(str))
	copy(*t, str)

	return nil
}

////////////////////////////////////////////////////////////////

type ValueBool bool

func newValueBool() Value {
	return new(ValueBool)
}

func (ValueBool) TypeID() byte {
	return 0x2
}

func (ValueBool) TypeString() string {
	return "Bool"
}

func (t *ValueBool) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t ValueBool) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(t.TypeID(), b, 1)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueBool) Bytes() []byte {
	if t {
		return []byte{0}
	} else {
		return []byte{1}
	}
}

func (t *ValueBool) FromBytes(b []byte) error {
	if len(b) != 1 {
		return errors.New("array length must be 1")
	}

	*t = b[0] != 0

	return nil
}

////////////////////////////////////////////////////////////////

type ValueInt int32

func newValueInt() Value {
	return new(ValueInt)
}

func (ValueInt) TypeID() byte {
	return 0x3
}

func (ValueInt) TypeString() string {
	return "Int"
}

func (t *ValueInt) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(t, a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t ValueInt) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	if err = deinterleave(bc, 4); err != nil {
		return nil, err
	}

	a, err = appendByteValues(t.TypeID(), bc, 4)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueInt) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, encodeZigzag(int32(t)))
	return b
}

func (t *ValueInt) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	v := binary.BigEndian.Uint32(b)
	*t = ValueInt(decodeZigzag(v))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueFloat float32

func newValueFloat() Value {
	return new(ValueFloat)
}

func (ValueFloat) TypeID() byte {
	return 0x4
}

func (ValueFloat) TypeString() string {
	return "Float"
}

func (t *ValueFloat) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(t, a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t ValueFloat) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	if err = deinterleave(bc, 4); err != nil {
		return nil, err
	}

	a, err = appendByteValues(t.TypeID(), bc, 4)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueFloat) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, encodeRobloxFloat(float32(t)))
	return b
}

func (t *ValueFloat) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	v := binary.BigEndian.Uint32(b)
	*t = ValueFloat(decodeRobloxFloat(v))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueDouble float64

func newValueDouble() Value {
	return new(ValueDouble)
}

func (ValueDouble) TypeID() byte {
	return 0x5
}

func (ValueDouble) TypeString() string {
	return "Double"
}

func (t *ValueDouble) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t ValueDouble) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(t.TypeID(), b, 4)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueDouble) Bytes() []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(math.Float64bits(float64(t))))
	return b
}

func (t *ValueDouble) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	v := binary.LittleEndian.Uint64(b)
	*t = ValueDouble(math.Float64frombits(v))

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

func (ValueUDim) TypeID() byte {
	return 0x6
}

func (ValueUDim) TypeString() string {
	return "UDim"
}

func (t *ValueUDim) ArrayBytes(a []Value) (b []byte, err error) {
	return nil, errors.New("not implemented")
}

func (t ValueUDim) FromArrayBytes(b []byte) (a []Value, err error) {
	return nil, errors.New("not implemented")
}

func (t ValueUDim) Bytes() []byte {
	b := make([]byte, 8)

	copy(b[0:4], t.Scale.Bytes())
	copy(b[4:8], t.Offset.Bytes())

	return b
}

func (t *ValueUDim) FromBytes(b []byte) error {
	if len(b) != 8 {
		return errors.New("array length must be 8")
	}

	t.Scale.FromBytes(b[0:4])
	t.Offset.FromBytes(b[4:8])

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

func (ValueUDim2) TypeID() byte {
	return 0x7
}

func (ValueUDim2) TypeString() string {
	return "UDim2"
}

func (t *ValueUDim2) ArrayBytes(a []Value) (b []byte, err error) {
	/*
		l := len(a)
		b = make([]byte, l*16)

		var n int
		for i, v := range a {
			vb := v.Bytes()

			for p := 0; p < 16; p += 4 {
				n = i*4 + l*p
				copy(b[n:n+4], vb[p:p+4])
			}
		}

		interleave(b, 4)
	*/

	b, err = appendValueBytes(t, a)

	// Interleave fields of each struct (field length, fields per struct).
	if err := bigInterleave(b, 4, 4); err != nil {
		return nil, err
	}

	// Interleave bytes of each field (byte length, bytes per field).
	if err := bigInterleave(b, 1, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t ValueUDim2) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)

	// Deinterleave bytes of each field (byte length, bytes per field).
	if err = bigDeinterleave(bc, 1, 4); err != nil {
		return nil, err
	}

	// Deinterleave fields of each struct (field length, fields per struct).
	if err = bigDeinterleave(bc, 4, 4); err != nil {
		return nil, err
	}

	a, err = appendByteValues(t.TypeID(), bc, 16)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueUDim2) Bytes() []byte {
	b := make([]byte, 16)
	copy(b[0:4], t.ScaleX.Bytes())
	copy(b[4:8], t.ScaleY.Bytes())
	copy(b[8:12], t.OffsetX.Bytes())
	copy(b[12:16], t.OffsetY.Bytes())
	return b
}

func (t *ValueUDim2) FromBytes(b []byte) error {
	if len(b) != 16 {
		return errors.New("array length must be 16")
	}

	t.ScaleX.FromBytes(b[0:4])
	t.ScaleY.FromBytes(b[4:8])
	t.OffsetX.FromBytes(b[8:12])
	t.OffsetY.FromBytes(b[12:16])

	return nil
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

func (ValueRay) TypeID() byte {
	return 0x8
}

func (ValueRay) TypeString() string {
	return "Ray"
}

func (t *ValueRay) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t ValueRay) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(t.TypeID(), b, 24)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueRay) Bytes() []byte {
	b := make([]byte, 24)
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(t.OriginX))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(t.OriginY))
	binary.LittleEndian.PutUint32(b[8:12], math.Float32bits(t.OriginZ))
	binary.LittleEndian.PutUint32(b[12:16], math.Float32bits(t.DirectionX))
	binary.LittleEndian.PutUint32(b[16:20], math.Float32bits(t.DirectionY))
	binary.LittleEndian.PutUint32(b[20:24], math.Float32bits(t.DirectionZ))
	return b
}

func (t *ValueRay) FromBytes(b []byte) error {
	if len(b) != 24 {
		return errors.New("array length must be 24")
	}

	t.OriginX = math.Float32frombits(binary.LittleEndian.Uint32(b[0:4]))
	t.OriginY = math.Float32frombits(binary.LittleEndian.Uint32(b[4:8]))
	t.OriginZ = math.Float32frombits(binary.LittleEndian.Uint32(b[8:12]))
	t.DirectionX = math.Float32frombits(binary.LittleEndian.Uint32(b[12:16]))
	t.DirectionY = math.Float32frombits(binary.LittleEndian.Uint32(b[16:20]))
	t.DirectionZ = math.Float32frombits(binary.LittleEndian.Uint32(b[20:24]))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueFaces struct {
	Right, Top, Back, Left, Bottom, Front bool
}

func newValueFaces() Value {
	return new(ValueFaces)
}

func (ValueFaces) TypeID() byte {
	return 0x9
}

func (ValueFaces) TypeString() string {
	return "Faces"
}

func (t *ValueFaces) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t ValueFaces) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(t.TypeID(), b, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueFaces) Bytes() []byte {
	flags := [6]bool{t.Front, t.Bottom, t.Left, t.Back, t.Top, t.Right}
	var b byte
	for i, flag := range flags {
		if flag {
			b = b | (1 << uint(i))
		}
	}

	return []byte{b}
}

func (t *ValueFaces) FromBytes(b []byte) error {
	if len(b) != 1 {
		return errors.New("array length must be 1")
	}

	t.Front = b[0]&(1<<0) != 0
	t.Bottom = b[0]&(1<<1) != 0
	t.Left = b[0]&(1<<2) != 0
	t.Back = b[0]&(1<<3) != 0
	t.Top = b[0]&(1<<4) != 0
	t.Right = b[0]&(1<<5) != 0

	return nil
}

////////////////////////////////////////////////////////////////

type ValueAxes struct {
	X, Y, Z bool
}

func newValueAxes() Value {
	return new(ValueAxes)
}

func (ValueAxes) TypeID() byte {
	return 0xA
}

func (ValueAxes) TypeString() string {
	return "Axes"
}

func (t *ValueAxes) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t ValueAxes) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(t.TypeID(), b, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueAxes) Bytes() []byte {
	flags := [3]bool{t.X, t.Y, t.Z}
	var b byte
	for i, flag := range flags {
		if flag {
			b = b | (1 << uint(i))
		}
	}

	return []byte{b}
}

func (t *ValueAxes) FromBytes(b []byte) error {
	if len(b) != 1 {
		return errors.New("array length must be 1")
	}

	t.X = b[0]&(1<<0) != 0
	t.Y = b[0]&(1<<1) != 0
	t.Z = b[0]&(1<<2) != 0

	return nil
}

////////////////////////////////////////////////////////////////

type ValueBrickColor uint32

func newValueBrickColor() Value {
	return new(ValueBrickColor)
}

func (ValueBrickColor) TypeID() byte {
	return 0xB
}

func (ValueBrickColor) TypeString() string {
	return "BrickColor"
}

func (t *ValueBrickColor) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(t, a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t ValueBrickColor) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	if err = deinterleave(bc, 4); err != nil {
		return nil, err
	}

	a, err = appendByteValues(t.TypeID(), bc, 4)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueBrickColor) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(t))
	return b
}

func (t *ValueBrickColor) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	*t = ValueBrickColor(binary.BigEndian.Uint32(b))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueColor3 struct {
	R, G, B ValueFloat
}

func newValueColor3() Value {
	return new(ValueColor3)
}

func (ValueColor3) TypeID() byte {
	return 0xC
}

func (ValueColor3) TypeString() string {
	return "Color3"
}

func (t *ValueColor3) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(t, a)

	// Interleave fields of each struct (field length, fields per struct).
	if err := bigInterleave(b, 4, 3); err != nil {
		return nil, err
	}

	// Interleave bytes of each field (byte length, bytes per field).
	if err := bigInterleave(b, 1, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t ValueColor3) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)

	// Deinterleave bytes of each field (byte length, bytes per field).
	if err = bigDeinterleave(bc, 1, 4); err != nil {
		return nil, err
	}

	// Deinterleave fields of each struct (field length, fields per struct).
	if err = bigDeinterleave(bc, 4, 3); err != nil {
		return nil, err
	}

	a, err = appendByteValues(t.TypeID(), bc, 12)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueColor3) Bytes() []byte {
	b := make([]byte, 12)
	copy(b[0:4], t.R.Bytes())
	copy(b[4:8], t.G.Bytes())
	copy(b[8:12], t.B.Bytes())
	return b
}

func (t *ValueColor3) FromBytes(b []byte) error {
	if len(b) != 12 {
		return errors.New("array length must be 12")
	}

	t.R.FromBytes(b[0:4])
	t.G.FromBytes(b[4:8])
	t.B.FromBytes(b[8:12])

	return nil
}

////////////////////////////////////////////////////////////////

type ValueVector2 struct {
	X, Y ValueFloat
}

func newValueVector2() Value {
	return new(ValueVector2)
}

func (ValueVector2) TypeID() byte {
	return 0xD
}

func (ValueVector2) TypeString() string {
	return "Vector2"
}

func (t *ValueVector2) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(t, a)

	// Interleave fields of each struct (field length, fields per struct).
	if err := bigInterleave(b, 4, 2); err != nil {
		return nil, err
	}

	// Interleave bytes of each field (byte length, bytes per field).
	if err := bigInterleave(b, 1, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t ValueVector2) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)

	// Deinterleave bytes of each field (byte length, bytes per field).
	if err = bigDeinterleave(bc, 1, 4); err != nil {
		return nil, err
	}

	// Deinterleave fields of each struct (field length, fields per struct).
	if err = bigDeinterleave(bc, 4, 2); err != nil {
		return nil, err
	}

	a, err = appendByteValues(t.TypeID(), bc, 8)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueVector2) Bytes() []byte {
	b := make([]byte, 8)
	copy(b[0:4], t.X.Bytes())
	copy(b[4:8], t.Y.Bytes())
	return b
}

func (t *ValueVector2) FromBytes(b []byte) error {
	if len(b) != 8 {
		return errors.New("array length must be 8")
	}

	t.X.FromBytes(b[0:4])
	t.Y.FromBytes(b[4:8])

	return nil
}

////////////////////////////////////////////////////////////////

type ValueVector3 struct {
	X, Y, Z ValueFloat
}

func newValueVector3() Value {
	return new(ValueVector3)
}

func (ValueVector3) TypeID() byte {
	return 0xE
}

func (ValueVector3) TypeString() string {
	return "Vector3"
}

func (t *ValueVector3) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(t, a)

	// Interleave fields of each struct (field length, fields per struct).
	if err := bigInterleave(b, 4, 3); err != nil {
		return nil, err
	}

	// Interleave bytes of each field (byte length, bytes per field).
	if err := bigInterleave(b, 1, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t ValueVector3) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)

	// Deinterleave bytes of each field (byte length, bytes per field).
	if err = bigDeinterleave(bc, 1, 4); err != nil {
		return nil, err
	}

	// Deinterleave fields of each struct (field length, fields per struct).
	if err = bigDeinterleave(bc, 4, 3); err != nil {
		return nil, err
	}

	a, err = appendByteValues(t.TypeID(), bc, 12)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueVector3) Bytes() []byte {
	b := make([]byte, 12)
	copy(b[0:4], t.X.Bytes())
	copy(b[4:8], t.Y.Bytes())
	copy(b[8:12], t.Z.Bytes())
	return b
}

func (t *ValueVector3) FromBytes(b []byte) error {
	if len(b) != 12 {
		return errors.New("array length must be 12")
	}

	t.X.FromBytes(b[0:4])
	t.Y.FromBytes(b[4:8])
	t.Z.FromBytes(b[8:12])

	return nil
}

////////////////////////////////////////////////////////////////

type ValueVector2int16 struct {
	X, Y int16
}

func newValueVector2int16() Value {
	return new(ValueVector2int16)
}

func (ValueVector2int16) TypeID() byte {
	return 0xF
}

func (ValueVector2int16) TypeString() string {
	return "Vector2int16"
}

func (t *ValueVector2int16) ArrayBytes(a []Value) (b []byte, err error) {
	return nil, errors.New("not implemented")
}

func (t ValueVector2int16) FromArrayBytes(b []byte) (a []Value, err error) {
	return nil, errors.New("not implemented")
}

func (t ValueVector2int16) Bytes() []byte {
	b := make([]byte, 4)

	binary.LittleEndian.PutUint16(b[0:2], uint16(t.X))
	binary.LittleEndian.PutUint16(b[2:4], uint16(t.Y))

	return b
}

func (t *ValueVector2int16) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	t.X = int16(binary.LittleEndian.Uint16(b[0:2]))
	t.Y = int16(binary.LittleEndian.Uint16(b[2:4]))

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

func (ValueCFrame) TypeID() byte {
	return 0x10
}

func (ValueCFrame) TypeString() string {
	return "CFrame"
}

func (t *ValueCFrame) ArrayBytes(a []Value) (b []byte, err error) {
	p := make([]Value, len(a))

	for i, v := range a {
		cf, ok := v.(*ValueCFrame)
		if !ok {
			return nil, errors.New(
				fmt.Sprintf("element %d is of type `%s` where `%s` is expected", i, v.TypeString(), t.TypeString()),
			)
		}

		// Build matrix part.
		b = append(b, cf.Special)
		if cf.Special == 0 {
			r := make([]byte, len(t.Rotation)*4)
			for i, f := range t.Rotation {
				binary.LittleEndian.PutUint32(r[i*4:i*4+4], math.Float32bits(f))
			}
			b = append(b, r...)
		}

		// Prepare position part.
		p[i] = &cf.Position
	}

	// Build position part.
	pb, _ := appendValueBytes(&t.Position, p)
	b = append(b, pb...)

	return b, nil
}

func (t ValueCFrame) FromArrayBytes(b []byte) (a []Value, err error) {
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
			q := len(t.Rotation) * 4
			r := b[i:]
			if len(r) < q {
				return nil, errors.New(fmt.Sprintf("expected %d more bytes in array", q))
			}
			for i := range t.Rotation {
				t.Rotation[i] = math.Float32frombits(binary.LittleEndian.Uint32(r[i*4 : i*4+4]))
			}
			i += q
		}

		cfs = append(cfs, cf)
	}

	// Read remaining position data using the Position field, which is a
	// ValueVector3. FromArrayBytes doesn't modify the value, so it's safe to
	// use from a non-pointer ValueVector3.
	a, err = t.Position.FromArrayBytes(b[i:])
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

func (t ValueCFrame) Bytes() []byte {
	var b []byte
	if t.Special == 0 {
		b = make([]byte, 49)
		r := b[1:]
		for i, f := range t.Rotation {
			binary.LittleEndian.PutUint32(r[i*4:i*4+4], math.Float32bits(f))
		}
	} else {
		b = make([]byte, 13)
		b[0] = t.Special
	}

	copy(b[len(b)-12:], t.Position.Bytes())

	return b
}

func (t *ValueCFrame) FromBytes(b []byte) error {
	if b[0] == 0 && len(b) != 49 {
		return errors.New("array length must be 49")
	} else if b[0] != 0 && len(b) != 13 {
		return errors.New("array length must be 13")
	}

	t.Special = b[0]

	if b[0] == 0 {
		r := b[1:]
		for i := range t.Rotation {
			t.Rotation[i] = math.Float32frombits(binary.LittleEndian.Uint32(r[i*4 : i*4+4]))
		}
	} else {
		for i := range t.Rotation {
			t.Rotation[i] = 0
		}
	}

	t.Position.FromBytes(b[len(b)-12:])

	return nil
}

////////////////////////////////////////////////////////////////

type ValueToken uint32

func newValueToken() Value {
	return new(ValueToken)
}

func (ValueToken) TypeID() byte {
	return 0x12
}

func (ValueToken) TypeString() string {
	return "Token"
}

func (t *ValueToken) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(t, a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t ValueToken) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	if err = deinterleave(bc, 4); err != nil {
		return nil, err
	}

	a, err = appendByteValues(t.TypeID(), bc, 4)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t ValueToken) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(t))
	return b
}

func (t *ValueToken) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	*t = ValueToken(binary.BigEndian.Uint32(b))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueReference int32

func newValueReference() Value {
	return new(ValueReference)
}

func (ValueReference) TypeID() byte {
	return 0x13
}

func (ValueReference) TypeString() string {
	return "Reference"
}

func (t *ValueReference) ArrayBytes(a []Value) (b []byte, err error) {
	if len(a) == 0 {
		return b, nil
	}

	size := 4
	b = make([]byte, len(a)*size)

	var prev ValueReference
	for i, v := range a {
		ref, ok := v.(*ValueReference)
		if !ok {
			return nil, errors.New(
				fmt.Sprintf("value %d is of type `%s` where `%s` is expected", i, v.TypeString(), t.TypeString()),
			)
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

func (t ValueReference) FromArrayBytes(b []byte) (a []Value, err error) {
	if len(b) == 0 {
		return a, nil
	}

	size := 4
	if len(b)%size != 0 {
		return nil, errors.New(fmt.Sprintf("array must be divisible by %d", size))
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
			*ref = *a[i-1].(*ValueReference) + *ref
		}

		a[i] = ref
	}

	return a, nil
}

func (t ValueReference) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, encodeZigzag(int32(t)))
	return b
}

func (t *ValueReference) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	v := binary.BigEndian.Uint32(b)
	*t = ValueReference(decodeZigzag(v))

	return nil
}

////////////////////////////////////////////////////////////////

type ValueVector3int16 struct {
	X, Y, Z int16
}

func newValueVector3int16() Value {
	return new(ValueVector3int16)
}

func (ValueVector3int16) TypeID() byte {
	return 0x14
}

func (ValueVector3int16) TypeString() string {
	return "Vector3int16"
}

func (t *ValueVector3int16) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t ValueVector3int16) FromArrayBytes(b []byte) (a []Value, err error) {
	return appendByteValues(t.TypeID(), b, 6)
}

func (t ValueVector3int16) Bytes() []byte {
	b := make([]byte, 6)

	binary.LittleEndian.PutUint16(b[0:2], uint16(t.X))
	binary.LittleEndian.PutUint16(b[2:4], uint16(t.Y))
	binary.LittleEndian.PutUint16(b[4:6], uint16(t.Z))

	return b
}

func (t *ValueVector3int16) FromBytes(b []byte) error {
	if len(b) != 6 {
		return errors.New("array length must be 6")
	}

	t.X = int16(binary.LittleEndian.Uint16(b[0:2]))
	t.Y = int16(binary.LittleEndian.Uint16(b[2:4]))
	t.Z = int16(binary.LittleEndian.Uint16(b[4:6]))

	return nil
}

////////////////////////////////////////////////////////////////
