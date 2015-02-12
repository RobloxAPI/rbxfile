package bin

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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
	newValueString().TypeID():     newValueString,
	newValueBool().TypeID():       newValueBool,
	newValueInt().TypeID():        newValueInt,
	newValueFloat().TypeID():      newValueFloat,
	newValueDouble().TypeID():     newValueDouble,
	newValueUDim().TypeID():       newValueUDim,
	newValueUDim2().TypeID():      newValueUDim2,
	newValueRay().TypeID():        newValueRay,
	newValueFaces().TypeID():      newValueFaces,
	newValueAxes().TypeID():       newValueAxes,
	newValueBrickColor().TypeID(): newValueBrickColor,
	newValueColor3().TypeID():     newValueColor3,
	newValueVector2().TypeID():    newValueVector2,
	newValueVector3().TypeID():    newValueVector3,
	//0xF: newValueVector2int16,
	//newValueCFrame().TypeID():     newValueCFrame,
	//0x11: newValueCFrameQuat,
	newValueToken().TypeID(): newValueToken,
	//newValueReferent().TypeID():   newValueReferent,
	newValueVector3int16().TypeID(): newValueVector3int16,
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
	r := bytes.NewReader(b)

	for r.Len() > 0 {
		var length uint32
		if err = binary.Read(r, binary.LittleEndian, &length); err == io.EOF {
			return nil, errors.New("unexpected EOF")
		}
		r.Seek(1, -4)

		vb := make([]byte, length+4)
		if _, err := r.Read(vb); err == io.EOF {
			return nil, errors.New("unexpected EOF")
		}

		var v *ValueString
		if err = v.FromBytes(vb); err != nil {
			return nil, err
		}
		a = append(a, v)
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
	if len(b)-4 >= 0 && length != uint32(len(b)-4) {
		return errors.New("string length does not match integer length")
	}

	*t = make(ValueString, len(b)-4)
	copy(*t, b)

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

	binary.LittleEndian.PutUint16(b[0:2], t.X)
	binary.LittleEndian.PutUint16(b[2:4], t.Y)
	binary.LittleEndian.PutUint16(b[4:6], t.Z)

	return b
}

func (t *ValueVector3int16) FromBytes(b []byte) error {
	if len(b) != 6 {
		return errors.New("array length must be 6")
	}

	t.X = binary.LittleEndian.Uint16(b[0:2])
	t.Y = binary.LittleEndian.Uint16(b[2:4])
	t.Z = binary.LittleEndian.Uint16(b[4:6])

	return nil
}

////////////////////////////////////////////////////////////////
