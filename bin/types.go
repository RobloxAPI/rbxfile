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
	newTypeString().TypeID(): newTypeString,
	newTypeBool().TypeID():   newTypeBool,
	newTypeInt().TypeID():    newTypeInt,
	newTypeFloat().TypeID():  newTypeFloat,
	newTypeDouble().TypeID(): newTypeDouble,
	//0x6: newTypeUDim,
	newTypeUDim2().TypeID(): newTypeUDim2,
	newTypeRay().TypeID():   newTypeRay,
	newTypeFaces().TypeID(): newTypeFaces,
	//newTypeAxes().TypeID():       newTypeAxes,
	//newTypeBrickColor().TypeID(): newTypeBrickColor,
	//newTypeColor3().TypeID():     newTypeColor3,
	//newTypeVector2().TypeID():    newTypeVector2,
	//newTypeVector3().TypeID():    newTypeVector3,
	//0xF: newTypeVector2int16,
	//newTypeCFrame().TypeID():     newTypeCFrame,
	//0x11: newTypeCFrameQuat,
	//newTypeToken().TypeID():      newTypeToken,
	//newTypeReferent().TypeID():   newTypeReferent,
	//0x14: newTypeVector3int16,
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

type TypeString []byte

func newTypeString() Value {
	return new(TypeString)
}

func (TypeString) TypeID() byte {
	return 0x1
}

func (TypeString) TypeString() string {
	return "String"
}

func (t *TypeString) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t TypeString) FromArrayBytes(b []byte) (a []Value, err error) {
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

		var v *TypeString
		if err = v.FromBytes(vb); err != nil {
			return nil, err
		}
		a = append(a, v)
	}

	return a, nil
}

func (t TypeString) Bytes() []byte {
	b := make([]byte, len(t)+4)
	binary.LittleEndian.PutUint32(b, uint32(len(t)))
	copy(b[4:], t)
	return b
}

func (t *TypeString) FromBytes(b []byte) error {
	if len(b) < 4 {
		return errors.New("array length must be greater than or equal to 4")
	}

	length := binary.LittleEndian.Uint32(b[:4])
	if len(b)-4 >= 0 && length != uint32(len(b)-4) {
		return errors.New("string length does not match integer length")
	}

	*t = make(TypeString, len(b)-4)
	copy(*t, b)

	return nil
}

////////////////////////////////////////////////////////////////

type TypeBool bool

func newTypeBool() Value {
	return new(TypeBool)
}

func (TypeBool) TypeID() byte {
	return 0x2
}

func (TypeBool) TypeString() string {
	return "Bool"
}

func (t *TypeBool) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t TypeBool) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(t.TypeID(), b, 1)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t TypeBool) Bytes() []byte {
	if t {
		return []byte{0}
	} else {
		return []byte{1}
	}
}

func (t *TypeBool) FromBytes(b []byte) error {
	if len(b) != 1 {
		return errors.New("array length must be 1")
	}

	*t = b[0] != 0

	return nil
}

////////////////////////////////////////////////////////////////

type TypeInt int32

func newTypeInt() Value {
	return new(TypeInt)
}

func (TypeInt) TypeID() byte {
	return 0x3
}

func (TypeInt) TypeString() string {
	return "Int"
}

func (t *TypeInt) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(t, a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t TypeInt) FromArrayBytes(b []byte) (a []Value, err error) {
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

func (t TypeInt) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, encodeZigzag(int32(t)))
	return b
}

func (t *TypeInt) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	v := binary.BigEndian.Uint32(b)
	*t = TypeInt(decodeZigzag(v))

	return nil
}

////////////////////////////////////////////////////////////////

type TypeFloat float32

func newTypeFloat() Value {
	return new(TypeFloat)
}

func (TypeFloat) TypeID() byte {
	return 0x4
}

func (TypeFloat) TypeString() string {
	return "Float"
}

func (t *TypeFloat) ArrayBytes(a []Value) (b []byte, err error) {
	b, err = appendValueBytes(t, a)
	if err != nil {
		return nil, err
	}

	if err = interleave(b, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t TypeFloat) FromArrayBytes(b []byte) (a []Value, err error) {
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

func (t TypeFloat) Bytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, encodeRobloxFloat(float32(t)))
	return b
}

func (t *TypeFloat) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	v := binary.BigEndian.Uint32(b)
	*t = TypeFloat(decodeRobloxFloat(v))

	return nil
}

////////////////////////////////////////////////////////////////

type TypeDouble float64

func newTypeDouble() Value {
	return new(TypeDouble)
}

func (TypeDouble) TypeID() byte {
	return 0x5
}

func (TypeDouble) TypeString() string {
	return "Double"
}

func (t *TypeDouble) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t TypeDouble) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(t.TypeID(), b, 4)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t TypeDouble) Bytes() []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(math.Float64bits(float64(t))))
	return b
}

func (t *TypeDouble) FromBytes(b []byte) error {
	if len(b) != 4 {
		return errors.New("array length must be 4")
	}

	v := binary.LittleEndian.Uint64(b)
	*t = TypeDouble(math.Float64frombits(v))

	return nil
}

////////////////////////////////////////////////////////////////

type TypeUDim2 struct {
	ScaleX  TypeFloat
	ScaleY  TypeFloat
	OffsetX TypeInt
	OffsetY TypeInt
}

func newTypeUDim2() Value {
	return new(TypeUDim2)
}

func (TypeUDim2) TypeID() byte {
	return 0x7
}

func (TypeUDim2) TypeString() string {
	return "UDim2"
}

func (t *TypeUDim2) ArrayBytes(a []Value) (b []byte, err error) {
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

	// Interleave fields of each UDim2 (field length, fields per UDim2).
	if err := bigInterleave(b, 4, 4); err != nil {
		return nil, err
	}

	// Interleave bytes of each field (byte length, bytes per field).
	if err := bigInterleave(b, 1, 4); err != nil {
		return nil, err
	}

	return b, nil
}

func (t TypeUDim2) FromArrayBytes(b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)

	// Deinterleave bytes of each field (byte length, bytes per field).
	if err = bigDeinterleave(bc, 1, 4); err != nil {
		return nil, err
	}

	// Deinterleave fields of each UDim2 (field length, fields per UDim2).
	if err = bigDeinterleave(bc, 4, 4); err != nil {
		return nil, err
	}

	a, err = appendByteValues(t.TypeID(), bc, 16)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t TypeUDim2) Bytes() []byte {
	b := make([]byte, 16)
	copy(b[0:4], t.ScaleX.Bytes())
	copy(b[4:8], t.ScaleY.Bytes())
	copy(b[8:12], t.OffsetX.Bytes())
	copy(b[12:16], t.OffsetY.Bytes())
	return b
}

func (t *TypeUDim2) FromBytes(b []byte) error {
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

type TypeRay struct {
	OriginX    float32
	OriginY    float32
	OriginZ    float32
	DirectionX float32
	DirectionY float32
	DirectionZ float32
}

func newTypeRay() Value {
	return new(TypeRay)
}

func (TypeRay) TypeID() byte {
	return 0x8
}

func (TypeRay) TypeString() string {
	return "Ray"
}

func (t *TypeRay) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t TypeRay) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(t.TypeID(), b, 24)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t TypeRay) Bytes() []byte {
	b := make([]byte, 24)
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(t.OriginX))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(t.OriginY))
	binary.LittleEndian.PutUint32(b[8:12], math.Float32bits(t.OriginZ))
	binary.LittleEndian.PutUint32(b[12:16], math.Float32bits(t.DirectionX))
	binary.LittleEndian.PutUint32(b[16:20], math.Float32bits(t.DirectionY))
	binary.LittleEndian.PutUint32(b[20:24], math.Float32bits(t.DirectionZ))
	return b
}

func (t *TypeRay) FromBytes(b []byte) error {
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

type TypeFaces struct {
	Right, Top, Back, Left, Bottom, Front bool
}

func newTypeFaces() Value {
	return new(TypeFaces)
}

func (TypeFaces) TypeID() byte {
	return 0x9
}

func (TypeFaces) TypeString() string {
	return "Faces"
}

func (t *TypeFaces) ArrayBytes(a []Value) (b []byte, err error) {
	return appendValueBytes(t, a)
}

func (t TypeFaces) FromArrayBytes(b []byte) (a []Value, err error) {
	a, err = appendByteValues(t.TypeID(), b, 0)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (t TypeFaces) Bytes() []byte {
	flags := [6]bool{t.Front, t.Bottom, t.Left, t.Back, t.Top, t.Right}
	var b byte
	for i, flag := range flags {
		if flag {
			b = b | (1 << uint(i))
		}
	}

	return []byte{b}
}

func (t *TypeFaces) FromBytes(b []byte) error {
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
