package rbxl

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	"github.com/robloxapi/rbxfile"
)

const (
	// Primitive sizes.
	zb   = 1 // byte
	zi8  = 1 // int8
	zu8  = 1 // uint8
	zi16 = 2 // int16
	zu16 = 2 // uint16
	zi32 = 4 // int32
	zu32 = 4 // uint32
	zf32 = 4 // float32
	zi64 = 8 // int64
	zu64 = 8 // uint64
	zf64 = 8 // float64

	// Variable size, where the type is an array of constant-size fields.
	zArray = -1

	// Variable size, where the size depends on the first decoded byte.
	zCond = -2

	// Variable size, where the type is optional.
	zOpt = -3

	// Number of bytes used to contain length of a zArray type.
	zArrayLen = 4

	// Number of bytes used for the condition of a zCond type.
	zCondLen = 1

	// Invalid size.
	zInvalid = 0
)

// typeID represents a type that can be serialized.
//
// Each type has a certain size indicating the number of bytes required to
// encode a value of the type.
//
// There are 4 kinds of type size: constant, array, conditional, and optional.
// Constant indicates that the size is constant, not depending on the content of
// the bytes. Array indicates an array of constant-sized fields, where the first
// zArrayLen bytes determines the length of the array. Conditional indicates
// that the size depends on the value of the first byte. Optional indicates that
// the type is the optional type, the size depending on the inner type.
type typeID byte

const (
	typeInvalid            typeID = 0x0
	typeString             typeID = 0x1
	typeBool               typeID = 0x2
	typeInt                typeID = 0x3
	typeFloat              typeID = 0x4
	typeDouble             typeID = 0x5
	typeUDim               typeID = 0x6
	typeUDim2              typeID = 0x7
	typeRay                typeID = 0x8
	typeFaces              typeID = 0x9
	typeAxes               typeID = 0xA
	typeBrickColor         typeID = 0xB
	typeColor3             typeID = 0xC
	typeVector2            typeID = 0xD
	typeVector3            typeID = 0xE
	typeVector2int16       typeID = 0xF
	typeCFrame             typeID = 0x10
	typeCFrameQuat         typeID = 0x11
	typeToken              typeID = 0x12
	typeReference          typeID = 0x13
	typeVector3int16       typeID = 0x14
	typeNumberSequence     typeID = 0x15
	typeColorSequence      typeID = 0x16
	typeNumberRange        typeID = 0x17
	typeRect               typeID = 0x18
	typePhysicalProperties typeID = 0x19
	typeColor3uint8        typeID = 0x1A
	typeInt64              typeID = 0x1B
	typeSharedString       typeID = 0x1C
	typeSignedString       typeID = 0x1D //TODO
	typeOptional           typeID = 0x1E
)

// Valid returns whether the type has a valid value.
func (t typeID) Valid() bool {
	return typeString <= t && t <= typeOptional && t != typeSignedString
}

// Size returns the number of bytes required to hold a value of the type.
// Returns < 0 if the size depends on the value, and 0 if the type is invalid.
//
// When < 0 is returned, the FieldSize or CondSize methods can be used to
// further determine the size.
func (t typeID) Size() int {
	switch t {
	case typeString:
		return zString
	case typeBool:
		return zBool
	case typeInt:
		return zInt
	case typeFloat:
		return zFloat
	case typeDouble:
		return zDouble
	case typeUDim:
		return zUDim
	case typeUDim2:
		return zUDim2
	case typeRay:
		return zRay
	case typeFaces:
		return zFaces
	case typeAxes:
		return zAxes
	case typeBrickColor:
		return zBrickColor
	case typeColor3:
		return zColor3
	case typeVector2:
		return zVector2
	case typeVector3:
		return zVector3
	case typeVector2int16:
		return zVector2int16
	case typeCFrame:
		return zCFrame
	case typeCFrameQuat:
		return zCFrameQuat
	case typeToken:
		return zToken
	case typeReference:
		return zReference
	case typeVector3int16:
		return zVector3int16
	case typeNumberSequence:
		return zNumberSequence
	case typeColorSequence:
		return zColorSequence
	case typeNumberRange:
		return zNumberRange
	case typeRect:
		return zRect
	case typePhysicalProperties:
		return zPhysicalProperties
	case typeColor3uint8:
		return zColor3uint8
	case typeInt64:
		return zInt64
	case typeSharedString:
		return zSharedString
	case typeOptional:
		return zOptional
	default:
		return zInvalid
	}
}

// FieldSize returns the byte size of each field within a value of the type,
// when the type's size is an array. Returns 0 if Size() does not return zArray.
func (t typeID) FieldSize() int {
	// Must return value that does not overflow uint32.
	switch t {
	case typeString:
		return zb
	case typeNumberSequence:
		return zNumberSequenceKeypoint
	case typeColorSequence:
		return zColorSequenceKeypoint
	default:
		return zInvalid
	}
}

// CondSize returns the byte size of the conditonal type t for condition b.
// Returns 0 if Size() does not return zCond. Note that the returned size
// includes the byte used as the condition.
func (t typeID) CondSize(b byte) int {
	switch t {
	case typeCFrame:
		if b == 0 {
			return zCFrameFull
		}
		return zCFrameShort
	case typeCFrameQuat:
		if b == 0 {
			return zCFrameQuatFull
		}
		return zCFrameQuatShort
	case typePhysicalProperties:
		if b == 0 {
			return zPhysicalPropertiesShort
		}
		return zPhysicalPropertiesFull
	}
	return zInvalid
}

// String returns a string representation of the type. If the type is not
// valid, then the returned value will be "Invalid".
func (t typeID) String() string {
	switch t {
	case typeString:
		return "String"
	case typeBool:
		return "Bool"
	case typeInt:
		return "Int"
	case typeFloat:
		return "Float"
	case typeDouble:
		return "Double"
	case typeUDim:
		return "UDim"
	case typeUDim2:
		return "UDim2"
	case typeRay:
		return "Ray"
	case typeFaces:
		return "Faces"
	case typeAxes:
		return "Axes"
	case typeBrickColor:
		return "BrickColor"
	case typeColor3:
		return "Color3"
	case typeVector2:
		return "Vector2"
	case typeVector3:
		return "Vector3"
	case typeVector2int16:
		return "Vector2int16"
	case typeCFrame:
		return "CFrame"
	case typeCFrameQuat:
		return "CFrameQuat"
	case typeToken:
		return "Token"
	case typeReference:
		return "Reference"
	case typeVector3int16:
		return "Vector3int16"
	case typeNumberSequence:
		return "NumberSequence"
	case typeColorSequence:
		return "ColorSequence"
	case typeNumberRange:
		return "NumberRange"
	case typeRect:
		return "Rect"
	case typePhysicalProperties:
		return "PhysicalProperties"
	case typeColor3uint8:
		return "Color3uint8"
	case typeInt64:
		return "Int64"
	case typeSharedString:
		return "SharedString"
	case typeOptional:
		return "Optional"
	default:
		return "Invalid"
	}
}

// ValueType returns the rbxfile.Type that corresponds to the type.
func (t typeID) ValueType() rbxfile.Type {
	switch t {
	case typeString:
		return rbxfile.TypeString
	case typeBool:
		return rbxfile.TypeBool
	case typeInt:
		return rbxfile.TypeInt
	case typeFloat:
		return rbxfile.TypeFloat
	case typeDouble:
		return rbxfile.TypeDouble
	case typeUDim:
		return rbxfile.TypeUDim
	case typeUDim2:
		return rbxfile.TypeUDim2
	case typeRay:
		return rbxfile.TypeRay
	case typeFaces:
		return rbxfile.TypeFaces
	case typeAxes:
		return rbxfile.TypeAxes
	case typeBrickColor:
		return rbxfile.TypeBrickColor
	case typeColor3:
		return rbxfile.TypeColor3
	case typeVector2:
		return rbxfile.TypeVector2
	case typeVector3:
		return rbxfile.TypeVector3
	case typeVector2int16:
		return rbxfile.TypeVector2int16
	case typeCFrame:
		return rbxfile.TypeCFrame
	case typeCFrameQuat:
		return rbxfile.TypeCFrame
	case typeToken:
		return rbxfile.TypeToken
	case typeReference:
		return rbxfile.TypeReference
	case typeVector3int16:
		return rbxfile.TypeVector3int16
	case typeNumberSequence:
		return rbxfile.TypeNumberSequence
	case typeColorSequence:
		return rbxfile.TypeColorSequence
	case typeNumberRange:
		return rbxfile.TypeNumberRange
	case typeRect:
		return rbxfile.TypeRect
	case typePhysicalProperties:
		return rbxfile.TypePhysicalProperties
	case typeColor3uint8:
		return rbxfile.TypeColor3uint8
	case typeInt64:
		return rbxfile.TypeInt64
	case typeSharedString:
		return rbxfile.TypeSharedString
	case typeOptional:
		return rbxfile.TypeOptional
	default:
		return rbxfile.TypeInvalid
	}
}

// fromValueType returns the Type corresponding to a given rbxfile.Type.
func fromValueType(t rbxfile.Type) typeID {
	switch t {
	case rbxfile.TypeString:
		return typeString
	case rbxfile.TypeBinaryString:
		return typeString
	case rbxfile.TypeProtectedString:
		return typeString
	case rbxfile.TypeContent:
		return typeString
	case rbxfile.TypeBool:
		return typeBool
	case rbxfile.TypeInt:
		return typeInt
	case rbxfile.TypeFloat:
		return typeFloat
	case rbxfile.TypeDouble:
		return typeDouble
	case rbxfile.TypeUDim:
		return typeUDim
	case rbxfile.TypeUDim2:
		return typeUDim2
	case rbxfile.TypeRay:
		return typeRay
	case rbxfile.TypeFaces:
		return typeFaces
	case rbxfile.TypeAxes:
		return typeAxes
	case rbxfile.TypeBrickColor:
		return typeBrickColor
	case rbxfile.TypeColor3:
		return typeColor3
	case rbxfile.TypeVector2:
		return typeVector2
	case rbxfile.TypeVector3:
		return typeVector3
	case rbxfile.TypeVector2int16:
		return typeVector2int16
	case rbxfile.TypeCFrame:
		return typeCFrame
	case rbxfile.TypeToken:
		return typeToken
	case rbxfile.TypeReference:
		return typeReference
	case rbxfile.TypeVector3int16:
		return typeVector3int16
	case rbxfile.TypeNumberSequence:
		return typeNumberSequence
	case rbxfile.TypeColorSequence:
		return typeColorSequence
	case rbxfile.TypeNumberRange:
		return typeNumberRange
	case rbxfile.TypeRect:
		return typeRect
	case rbxfile.TypePhysicalProperties:
		return typePhysicalProperties
	case rbxfile.TypeColor3uint8:
		return typeColor3uint8
	case rbxfile.TypeInt64:
		return typeInt64
	case rbxfile.TypeSharedString:
		return typeSharedString
	case rbxfile.TypeOptional:
		return typeOptional
	default:
		return typeInvalid
	}
}

// value represents a value of a certain Type.
type value interface {
	// Type returns an identifier indicating the type.
	Type() typeID

	// BytesLen returns the number of bytes required to encode the value.
	BytesLen() int

	// Bytes encodes value to b, returning the extended buffer.
	Bytes(b []byte) []byte

	// FromBytes decodes the value from b. Returns an error if the value could
	// not be decoded. Otherwise, returns the number of bytes read from b.
	FromBytes(b []byte) (n int, err error)

	Dump(bw *bufio.Writer, indent int)
}

// newValue returns new Value of the given Type. The initial value will not
// necessarily be the zero for the type. If the given type is invalid, then a
// nil value is returned.
func newValue(typ typeID) value {
	switch typ {
	case typeString:
		return new(valueString)
	case typeBool:
		return new(valueBool)
	case typeInt:
		return new(valueInt)
	case typeFloat:
		return new(valueFloat)
	case typeDouble:
		return new(valueDouble)
	case typeUDim:
		return new(valueUDim)
	case typeUDim2:
		return new(valueUDim2)
	case typeRay:
		return new(valueRay)
	case typeFaces:
		return new(valueFaces)
	case typeAxes:
		return new(valueAxes)
	case typeBrickColor:
		return new(valueBrickColor)
	case typeColor3:
		return new(valueColor3)
	case typeVector2:
		return new(valueVector2)
	case typeVector3:
		return new(valueVector3)
	case typeVector2int16:
		return new(valueVector2int16)
	case typeCFrame:
		return new(valueCFrame)
	case typeCFrameQuat:
		return new(valueCFrameQuat)
	case typeToken:
		return new(valueToken)
	case typeReference:
		return new(valueReference)
	case typeVector3int16:
		return new(valueVector3int16)
	case typeNumberSequence:
		return new(valueNumberSequence)
	case typeColorSequence:
		return new(valueColorSequence)
	case typeNumberRange:
		return new(valueNumberRange)
	case typeRect:
		return new(valueRect)
	case typePhysicalProperties:
		return new(valuePhysicalProperties)
	case typeColor3uint8:
		return new(valueColor3uint8)
	case typeInt64:
		return new(valueInt64)
	case typeSharedString:
		return new(valueSharedString)
	case typeOptional:
		return new(valueOptional)
	}
	return nil
}

////////////////////////////////////////////////////////////////

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

type buflenError struct {
	exp uint64
	got int
}

func (err buflenError) Error() string {
	return fmt.Sprintf("expected %d bytes, got %d", err.exp, err.got)
}

// checkLengthConst does a basic check of the buffer's length against a value
// with a constant expected byte length. Returns the number of bytes expected,
// and an error if len(b) is less than n.
func checkLengthConst(v interface {
	Type() typeID
	BytesLen() int
}, b []byte) (n int, err error) {
	if n = v.BytesLen(); len(b) < n {
		return n, buflenError{exp: uint64(n), got: len(b)}
	}
	return n, nil
}

// checkLengthArray checks the number of bytes required to decode v from b,
// where v is assumed to be of a type where Type().Size() returns a zArray.
// Returns the number of fields in the array and the remaining buffer. Returns
// an error if the buffer is too short.
//
// At least zArrayLen bytes are required, which are decoded as the length of the
// array. The remaining buffer is expected to be v.Type().FieldSize()*length in
// length.
func checkLengthArray(v value, b []byte) (r []byte, n int, err error) {
	if len(b) < zArrayLen {
		return b, 0, buflenError{exp: zArrayLen, got: len(b)}
	}
	length := binary.LittleEndian.Uint32(b[:zArrayLen])
	if n := zArrayLen + uint64(v.Type().FieldSize())*uint64(length); uint64(len(b)) < n {
		return b, 0, buflenError{exp: n, got: len(b)}
	}
	return b[zArrayLen:], int(length), nil
}

// checkLengthCond returns the number of bytes required to decode v from b,
// where v is assumed to be of a type where Type().Size() returns zCond.
//
// At least 1 byte is required, which is decoded as the condition.
func checkLengthCond(v value, b []byte) (cond byte, r []byte, n int, err error) {
	if len(b) < zCondLen {
		return cond, b, 0, buflenError{exp: zCondLen, got: len(b)}
	}
	cond = b[0]
	if n = v.Type().CondSize(cond); len(b) < n {
		return cond, b[zCondLen:], zCondLen, buflenError{exp: uint64(n), got: len(b)}
	}
	return cond, b[zCondLen:], n, nil
}

var le = binary.LittleEndian
var be = binary.BigEndian

// appendUint16 appends v to b in the given byte order, returning the extended
// buffer.
func appendUint16(b []byte, order binary.ByteOrder, v uint16) []byte {
	var a [zu16]byte
	order.PutUint16(a[:], v)
	return append(b, a[:]...)
}

// appendUint32 appends v to b in the given byte order, returning the extended
// buffer.
func appendUint32(b []byte, order binary.ByteOrder, v uint32) []byte {
	var a [zu32]byte
	order.PutUint32(a[:], v)
	return append(b, a[:]...)
}

// appendUint64 appends v to b in the given byte order, returning the extended
// buffer.
func appendUint64(b []byte, order binary.ByteOrder, v uint64) []byte {
	var a [zu64]byte
	order.PutUint64(a[:], v)
	return append(b, a[:]...)
}

// appendFlags appends to b up to 8 flags as one byte.
func appendFlags(b []byte, flags ...bool) []byte {
	var a byte
	for i, flag := range flags {
		if flag {
			a |= 1 << uint(i)
		}
	}
	return append(b, a)
}

func readFlags(b byte, flags ...*bool) {
	for i, f := range flags {
		*f = b&(1<<i) != 0
	}
}

// readUint8 decodes a uint8 from b, then advances b by the size of the value.
func readUint8(b *[]byte) uint8 {
	n := (*b)[0]
	*b = (*b)[zu8:]
	return n
}

// readUint16 decodes a uint16 from b in the given byte order, then advances b
// by the size of the value.
func readUint16(b *[]byte, order binary.ByteOrder) uint16 {
	n := order.Uint16(*b)
	*b = (*b)[zu16:]
	return n
}

// readUint32 decodes a uint32 from b in the given byte order, then advances b
// by the size of the value.
func readUint32(b *[]byte, order binary.ByteOrder) uint32 {
	n := order.Uint32(*b)
	*b = (*b)[zu32:]
	return n
}

// readUint64 decodes a uint64 from b in the given byte order, then advances b
// by the size of the value.
func readUint64(b *[]byte, order binary.ByteOrder) uint64 {
	n := order.Uint64(*b)
	*b = (*b)[zu64:]
	return n
}

// fromBytes calls v.FromBytes(b), then returns b advanced by n bytes. Panics if
// v.FromBytes returns an error.
func fromBytes(b []byte, v value) []byte {
	n, err := v.FromBytes(b)
	if err != nil {
		panic(err)
	}
	return b[n:]
}

////////////////////////////////////////////////////////////////

const zString = zArray

type valueString []byte

func (valueString) Type() typeID {
	return typeString
}

func (v valueString) BytesLen() int {
	return zArrayLen + len(v)
}

func (v valueString) Bytes(b []byte) []byte {
	b = appendUint32(b, le, uint32(len(v)))
	b = append(b, v...)
	return b
}

func (v *valueString) FromBytes(b []byte) (n int, err error) {
	if b, n, err = checkLengthArray(v, b); err != nil {
		return n, err
	}
	*v = make(valueString, n)
	copy(*v, b)
	return v.BytesLen(), nil
}

func (v valueString) Dump(w *bufio.Writer, indent int) {
	dumpString(w, indent, string(v))
}

////////////////////////////////////////////////////////////////

const zBool = zb

type valueBool bool

func (valueBool) Type() typeID {
	return typeBool
}

func (v valueBool) BytesLen() int {
	return zBool
}

func (v valueBool) Bytes(b []byte) []byte {
	if v {
		return append(b, 1)
	} else {
		return append(b, 0)
	}
}

func (v *valueBool) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	*v = b[0] != 0
	return n, nil
}

func (v valueBool) Dump(w *bufio.Writer, indent int) {
	if v {
		w.WriteString("true")
	} else {
		w.WriteString("false")
	}
}

////////////////////////////////////////////////////////////////

const zInt = zi32

type valueInt int32

func (valueInt) Type() typeID {
	return typeInt
}

func (v valueInt) BytesLen() int {
	return zInt
}

func (v valueInt) Bytes(b []byte) []byte {
	return appendUint32(b, be, encodeZigzag32(int32(v)))
}

func (v *valueInt) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	*v = valueInt(decodeZigzag32(binary.BigEndian.Uint32(b)))
	return n, nil
}

func (v valueInt) Dump(w *bufio.Writer, indent int) {
	w.Write(strconv.AppendInt(nil, int64(v), 10))
}

////////////////////////////////////////////////////////////////

const zFloat = zf32

type valueFloat float32

func (valueFloat) Type() typeID {
	return typeFloat
}

func (v valueFloat) BytesLen() int {
	return zFloat
}

func (v valueFloat) Bytes(b []byte) []byte {
	return appendUint32(b, be, encodeRobloxFloat(float32(v)))
}

func (v *valueFloat) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	*v = valueFloat(decodeRobloxFloat(binary.BigEndian.Uint32(b)))
	return n, nil
}

func (v valueFloat) Dump(w *bufio.Writer, indent int) {
	w.Write(strconv.AppendFloat(nil, float64(v), 'g', -1, 32))
}

////////////////////////////////////////////////////////////////

const zDouble = zf64

type valueDouble float64

func (valueDouble) Type() typeID {
	return typeDouble
}

func (v valueDouble) BytesLen() int {
	return zDouble
}

func (v valueDouble) Bytes(b []byte) []byte {
	return appendUint64(b, le, math.Float64bits(float64(v)))
}

func (v *valueDouble) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	*v = valueDouble(math.Float64frombits(binary.LittleEndian.Uint64(b)))
	return n, nil
}

func (v valueDouble) Dump(w *bufio.Writer, indent int) {
	w.Write(strconv.AppendFloat(nil, float64(v), 'g', -1, 64))
}

////////////////////////////////////////////////////////////////

const zUDim = zFloat + zInt

type valueUDim struct {
	Scale  valueFloat
	Offset valueInt
}

func (valueUDim) Type() typeID {
	return typeUDim
}

func (v valueUDim) BytesLen() int {
	return zUDim
}

func (v valueUDim) Bytes(b []byte) []byte {
	b = v.Scale.Bytes(b)
	b = v.Offset.Bytes(b)
	return b
}

func (v *valueUDim) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	b = fromBytes(b, &v.Scale)
	b = fromBytes(b, &v.Offset)
	return n, nil
}

func (v valueUDim) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("Scale: ")
	v.Scale.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Offset: ")
	v.Offset.Dump(w, indent+1)

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zUDim2 = zFloat*2 + zInt*2

type valueUDim2 struct {
	ScaleX  valueFloat
	ScaleY  valueFloat
	OffsetX valueInt
	OffsetY valueInt
}

func (valueUDim2) Type() typeID {
	return typeUDim2
}

func (v valueUDim2) BytesLen() int {
	return zUDim2
}

func (v valueUDim2) Bytes(b []byte) []byte {
	b = v.ScaleX.Bytes(b)
	b = v.ScaleY.Bytes(b)
	b = v.OffsetX.Bytes(b)
	b = v.OffsetY.Bytes(b)
	return b
}

func (v *valueUDim2) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	b = fromBytes(b, &v.ScaleX)
	b = fromBytes(b, &v.ScaleY)
	b = fromBytes(b, &v.OffsetX)
	b = fromBytes(b, &v.OffsetY)
	return n, nil
}

func (v valueUDim2) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("X.Scale: ")
	v.ScaleX.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Y.Scale: ")
	v.OffsetX.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("X.Offset: ")
	v.ScaleY.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Y.Offset: ")
	v.OffsetY.Dump(w, indent+1)

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zRay = zf32 * 6

type valueRay struct {
	OriginX    float32
	OriginY    float32
	OriginZ    float32
	DirectionX float32
	DirectionY float32
	DirectionZ float32
}

func (valueRay) Type() typeID {
	return typeRay
}

func (v valueRay) BytesLen() int {
	return zRay
}

func (v valueRay) Bytes(b []byte) []byte {
	b = appendUint32(b, le, math.Float32bits(v.OriginX))
	b = appendUint32(b, le, math.Float32bits(v.OriginY))
	b = appendUint32(b, le, math.Float32bits(v.OriginZ))
	b = appendUint32(b, le, math.Float32bits(v.DirectionX))
	b = appendUint32(b, le, math.Float32bits(v.DirectionY))
	b = appendUint32(b, le, math.Float32bits(v.DirectionZ))
	return b
}

func (v *valueRay) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	v.OriginX = math.Float32frombits(readUint32(&b, le))
	v.OriginY = math.Float32frombits(readUint32(&b, le))
	v.OriginZ = math.Float32frombits(readUint32(&b, le))
	v.DirectionX = math.Float32frombits(readUint32(&b, le))
	v.DirectionY = math.Float32frombits(readUint32(&b, le))
	v.DirectionZ = math.Float32frombits(readUint32(&b, le))
	return n, nil
}

func (v valueRay) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("Origin.X: ")
	w.Write(strconv.AppendFloat(nil, float64(v.OriginX), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("Origin.Y: ")
	w.Write(strconv.AppendFloat(nil, float64(v.OriginY), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("Origin.Z: ")
	w.Write(strconv.AppendFloat(nil, float64(v.OriginZ), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("Direction.X: ")
	w.Write(strconv.AppendFloat(nil, float64(v.DirectionX), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("Direction.Y: ")
	w.Write(strconv.AppendFloat(nil, float64(v.DirectionY), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("Direction.Z: ")
	w.Write(strconv.AppendFloat(nil, float64(v.DirectionZ), 'g', -1, 32))

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zFaces = zu8

type valueFaces struct {
	Right, Top, Back, Left, Bottom, Front bool
}

func (valueFaces) Type() typeID {
	return typeFaces
}

func (v valueFaces) BytesLen() int {
	return zFaces
}

func (v valueFaces) Bytes(b []byte) []byte {
	return appendFlags(b, v.Right, v.Top, v.Back, v.Left, v.Bottom, v.Front)
}

func (v *valueFaces) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	readFlags(b[0], &v.Right, &v.Top, &v.Back, &v.Left, &v.Bottom, &v.Front)
	return n, nil
}

func (v valueFaces) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("Right: ")
	w.Write(strconv.AppendBool(nil, v.Right))

	dumpNewline(w, indent+1)
	w.WriteString("Top: ")
	w.Write(strconv.AppendBool(nil, v.Top))

	dumpNewline(w, indent+1)
	w.WriteString("Back: ")
	w.Write(strconv.AppendBool(nil, v.Back))

	dumpNewline(w, indent+1)
	w.WriteString("Left: ")
	w.Write(strconv.AppendBool(nil, v.Left))

	dumpNewline(w, indent+1)
	w.WriteString("Bottom: ")
	w.Write(strconv.AppendBool(nil, v.Bottom))

	dumpNewline(w, indent+1)
	w.WriteString("Front: ")
	w.Write(strconv.AppendBool(nil, v.Front))

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zAxes = zu8

type valueAxes struct {
	X, Y, Z bool
}

func (valueAxes) Type() typeID {
	return typeAxes
}

func (v valueAxes) BytesLen() int {
	return zAxes
}

func (v valueAxes) Bytes(b []byte) []byte {
	return appendFlags(b, v.X, v.Y, v.Z)
}

func (v *valueAxes) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	readFlags(b[0], &v.X, &v.Y, &v.Z)
	return n, nil
}

func (v valueAxes) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("X: ")
	w.Write(strconv.AppendBool(nil, v.X))

	dumpNewline(w, indent+1)
	w.WriteString("Y: ")
	w.Write(strconv.AppendBool(nil, v.Y))

	dumpNewline(w, indent+1)
	w.WriteString("Z: ")
	w.Write(strconv.AppendBool(nil, v.Z))

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zBrickColor = zu32

type valueBrickColor uint32

func (valueBrickColor) Type() typeID {
	return typeBrickColor
}

func (v valueBrickColor) BytesLen() int {
	return zBrickColor
}

func (v valueBrickColor) Bytes(b []byte) []byte {
	return appendUint32(b, be, uint32(v))
}

func (v *valueBrickColor) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	*v = valueBrickColor(binary.BigEndian.Uint32(b))
	return n, nil
}

func (v valueBrickColor) Dump(w *bufio.Writer, indent int) {
	w.Write(strconv.AppendUint(nil, uint64(v), 10))
}

////////////////////////////////////////////////////////////////

const zColor3 = zFloat * 3

type valueColor3 struct {
	R, G, B valueFloat
}

func (valueColor3) Type() typeID {
	return typeColor3
}

func (v valueColor3) BytesLen() int {
	return zColor3
}

func (v valueColor3) Bytes(b []byte) []byte {
	b = v.R.Bytes(b)
	b = v.G.Bytes(b)
	b = v.B.Bytes(b)
	return b
}

func (v *valueColor3) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	b = fromBytes(b, &v.R)
	b = fromBytes(b, &v.G)
	b = fromBytes(b, &v.B)
	return n, nil
}

func (v valueColor3) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("R: ")
	v.R.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("G: ")
	v.G.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("B: ")
	v.B.Dump(w, indent+1)

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zVector2 = zFloat * 2

type valueVector2 struct {
	X, Y valueFloat
}

func (valueVector2) Type() typeID {
	return typeVector2
}

func (v valueVector2) BytesLen() int {
	return zVector2
}

func (v valueVector2) Bytes(b []byte) []byte {
	b = v.X.Bytes(b)
	b = v.Y.Bytes(b)
	return b
}

func (v *valueVector2) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	b = fromBytes(b, &v.X)
	b = fromBytes(b, &v.Y)
	return n, nil
}

func (v valueVector2) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("X: ")
	v.X.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Y: ")
	v.Y.Dump(w, indent+1)

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zVector3 = zFloat * 3

type valueVector3 struct {
	X, Y, Z valueFloat
}

func (valueVector3) Type() typeID {
	return typeVector3
}

func (v valueVector3) BytesLen() int {
	return zVector3
}

func (v valueVector3) Bytes(b []byte) []byte {
	b = v.X.Bytes(b)
	b = v.Y.Bytes(b)
	b = v.Z.Bytes(b)
	return b
}

func (v *valueVector3) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	b = fromBytes(b, &v.X)
	b = fromBytes(b, &v.Y)
	b = fromBytes(b, &v.Z)
	return n, nil
}

func (v valueVector3) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("X: ")
	v.X.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Y: ")
	v.Y.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Z: ")
	v.Z.Dump(w, indent+1)

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zVector2int16 = zi16 * 2

type valueVector2int16 struct {
	X, Y int16
}

func (valueVector2int16) Type() typeID {
	return typeVector2int16
}

func (v valueVector2int16) BytesLen() int {
	return zVector2int16
}

func (v valueVector2int16) Bytes(b []byte) []byte {
	b = appendUint16(b, le, uint16(v.X))
	b = appendUint16(b, le, uint16(v.Y))
	return b
}

func (v *valueVector2int16) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	v.X = int16(readUint16(&b, le))
	v.Y = int16(readUint16(&b, le))
	return n, nil
}

func (v valueVector2int16) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("X: ")
	w.Write(strconv.AppendInt(nil, int64(v.X), 10))

	dumpNewline(w, indent+1)
	w.WriteString("Y: ")
	w.Write(strconv.AppendInt(nil, int64(v.Y), 10))

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zCFrame = zCond
const zCFrameSp = zCondLen
const zCFrameRo = zf32 * 9
const zCFrameFull = zCFrameSp + zCFrameRo + zVector3
const zCFrameShort = zCFrameSp + zVector3

type valueCFrame struct {
	Special  uint8
	Rotation [9]float32
	Position valueVector3
}

func (valueCFrame) Type() typeID {
	return typeCFrame
}

func (v valueCFrame) BytesLen() int {
	return v.Type().CondSize(v.Special)
}

func (v valueCFrame) Bytes(b []byte) []byte {
	b = append(b, v.Special)
	if v.Special == 0 {
		for _, f := range v.Rotation {
			b = appendUint32(b, le, math.Float32bits(f))
		}
	}
	b = v.Position.Bytes(b)
	return b
}

func (v *valueCFrame) FromBytes(b []byte) (n int, err error) {
	cond, b, n, err := checkLengthCond(v, b)
	if err != nil {
		return n, err
	}
	v.Special = cond
	if cond == 0 {
		for i := range v.Rotation {
			v.Rotation[i] = math.Float32frombits(readUint32(&b, le))
		}
	} else {
		for i := range v.Rotation {
			v.Rotation[i] = 0
		}
	}
	v.Position.FromBytes(b)
	return n, nil
}

func (v valueCFrame) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("ID: ")
	w.Write(strconv.AppendUint(nil, uint64(v.Special), 10))

	if v.Special == 0 {
		dumpNewline(w, indent+1)
		w.WriteString("Rotation: {")
		for _, r := range v.Rotation {
			dumpNewline(w, indent+2)
			w.Write(strconv.AppendFloat(nil, float64(r), 'g', -1, 32))
		}
		dumpNewline(w, indent+1)
		w.WriteByte('}')
	}

	dumpNewline(w, indent+1)
	w.WriteString("Position: ")
	v.Position.Dump(w, indent+1)

	dumpNewline(w, indent)
	w.WriteByte('}')
}

func sqrt32(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

// ToCFrameQuat converts the value to a valueCFrameQuat.
func (v valueCFrame) ToCFrameQuat() (q valueCFrameQuat) {
	if v.Special != 0 {
		return valueCFrameQuat{
			Special:  v.Special,
			Position: v.Position,
		}
	}
	q.Position = v.Position
	if v.Rotation[0]+v.Rotation[4]+v.Rotation[8] > 0 {
		q.QW = sqrt32(1+v.Rotation[0]+v.Rotation[4]+v.Rotation[8]) * 0.5
		q.QX = (v.Rotation[7] - v.Rotation[5]) / (4 * q.QW)
		q.QY = (v.Rotation[2] - v.Rotation[6]) / (4 * q.QW)
		q.QZ = (v.Rotation[3] - v.Rotation[1]) / (4 * q.QW)
	} else if v.Rotation[0] > v.Rotation[4] && v.Rotation[0] > v.Rotation[8] {
		q.QX = sqrt32(1+v.Rotation[0]-v.Rotation[4]-v.Rotation[8]) * 0.5
		q.QY = (v.Rotation[3] + v.Rotation[1]) / (4 * q.QX)
		q.QZ = (v.Rotation[6] + v.Rotation[2]) / (4 * q.QX)
		q.QW = (v.Rotation[7] - v.Rotation[5]) / (4 * q.QX)
	} else if v.Rotation[4] > v.Rotation[8] {
		q.QY = sqrt32(1+v.Rotation[4]-v.Rotation[0]-v.Rotation[8]) * 0.5
		q.QX = (v.Rotation[3] + v.Rotation[1]) / (4 * q.QY)
		q.QZ = (v.Rotation[7] + v.Rotation[5]) / (4 * q.QY)
		q.QW = (v.Rotation[2] - v.Rotation[6]) / (4 * q.QY)
	} else {
		q.QZ = sqrt32(1+v.Rotation[8]-v.Rotation[0]-v.Rotation[4]) * 0.5
		q.QX = (v.Rotation[6] + v.Rotation[2]) / (4 * q.QZ)
		q.QY = (v.Rotation[7] + v.Rotation[5]) / (4 * q.QZ)
		q.QW = (v.Rotation[3] - v.Rotation[1]) / (4 * q.QZ)
	}
	return q
}

////////////////////////////////////////////////////////////////

const zCFrameQuat = zCond
const zCFrameQuatSp = zCondLen
const zCFrameQuatQ = zf32 * 4
const zCFrameQuatFull = zCFrameQuatSp + zCFrameQuatQ + zVector3
const zCFrameQuatShort = zCFrameQuatSp + zVector3

type valueCFrameQuat struct {
	Special        uint8
	QX, QY, QZ, QW float32
	Position       valueVector3
}

func (valueCFrameQuat) Type() typeID {
	return typeCFrameQuat
}

func (v valueCFrameQuat) BytesLen() int {
	return v.Type().CondSize(v.Special)
}

func (v valueCFrameQuat) quatBytes(b []byte) []byte {
	b = appendUint32(b, le, math.Float32bits(v.QX))
	b = appendUint32(b, le, math.Float32bits(v.QY))
	b = appendUint32(b, le, math.Float32bits(v.QZ))
	b = appendUint32(b, le, math.Float32bits(v.QW))
	return b
}

func (v valueCFrameQuat) Bytes(b []byte) []byte {
	b = append(b, v.Special)
	if v.Special == 0 {
		b = v.quatBytes(b)
	}
	b = v.Position.Bytes(b)
	return b
}

func (v *valueCFrameQuat) quatFromBytes(b []byte) []byte {
	v.QX = math.Float32frombits(readUint32(&b, le))
	v.QY = math.Float32frombits(readUint32(&b, le))
	v.QZ = math.Float32frombits(readUint32(&b, le))
	v.QW = math.Float32frombits(readUint32(&b, le))
	return b
}

func (v *valueCFrameQuat) FromBytes(b []byte) (n int, err error) {
	cond, b, n, err := checkLengthCond(v, b)
	if err != nil {
		return n, err
	}
	v.Special = cond
	if cond == 0 {
		b = v.quatFromBytes(b)
	} else {
		v.QX = 0
		v.QY = 0
		v.QZ = 0
		v.QW = 0
	}
	v.Position.FromBytes(b)
	return n, nil
}

func (v valueCFrameQuat) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("ID: ")
	w.Write(strconv.AppendUint(nil, uint64(v.Special), 10))

	dumpNewline(w, indent+1)
	w.WriteString("QX: ")
	w.Write(strconv.AppendFloat(nil, float64(v.QX), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("QY: ")
	w.Write(strconv.AppendFloat(nil, float64(v.QY), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("QZ: ")
	w.Write(strconv.AppendFloat(nil, float64(v.QZ), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("QW: ")
	w.Write(strconv.AppendFloat(nil, float64(v.QW), 'g', -1, 32))

	w.WriteString("Position: ")
	v.Position.Dump(w, indent+1)

	dumpNewline(w, indent)
	w.WriteByte('}')
}

// ToCFrame converts the value to a valueCFrame.
func (v valueCFrameQuat) ToCFrame() valueCFrame {
	if v.Special != 0 {
		return valueCFrame{
			Special:  v.Special,
			Position: v.Position,
		}
	}
	return valueCFrame{
		Position: v.Position,
		Rotation: [9]float32{
			1 - 2*(v.QY*v.QY+v.QZ*v.QZ), 2 * (v.QY*v.QX - v.QW*v.QZ), 2 * (v.QW*v.QY + v.QZ*v.QX),
			2 * (v.QW*v.QZ + v.QY*v.QX), 1 - 2*(v.QX*v.QX+v.QZ*v.QZ), 2 * (v.QZ*v.QY - v.QW*v.QX),
			2 * (v.QZ*v.QX - v.QW*v.QY), 2 * (v.QW*v.QX + v.QZ*v.QY), 1 - 2*(v.QX*v.QX+v.QY*v.QY),
		},
	}
}

////////////////////////////////////////////////////////////////

const zToken = zu32

type valueToken uint32

func (valueToken) Type() typeID {
	return typeToken
}

func (v valueToken) BytesLen() int {
	return zToken
}

func (v valueToken) Bytes(b []byte) []byte {
	return appendUint32(b, be, uint32(v))
}

func (v *valueToken) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	*v = valueToken(binary.BigEndian.Uint32(b))
	return n, nil
}

func (v valueToken) Dump(w *bufio.Writer, indent int) {
	w.Write(strconv.AppendUint(nil, uint64(v), 10))
}

////////////////////////////////////////////////////////////////

const zReference = zi32

type valueReference int32

func (valueReference) Type() typeID {
	return typeReference
}

func (v valueReference) BytesLen() int {
	return zReference
}

func (v valueReference) Bytes(b []byte) []byte {
	return appendUint32(b, be, encodeZigzag32(int32(v)))
}

func (v *valueReference) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	*v = valueReference(decodeZigzag32(binary.BigEndian.Uint32(b)))
	return n, nil
}

func (v valueReference) Dump(w *bufio.Writer, indent int) {
	w.Write(strconv.AppendInt(nil, int64(v), 10))
}

////////////////////////////////////////////////////////////////

const zVector3int16 = zi16 * 3

type valueVector3int16 struct {
	X, Y, Z int16
}

func (valueVector3int16) Type() typeID {
	return typeVector3int16
}

func (v valueVector3int16) BytesLen() int {
	return zVector3int16
}

func (v valueVector3int16) Bytes(b []byte) []byte {
	b = appendUint16(b, le, uint16(v.X))
	b = appendUint16(b, le, uint16(v.Y))
	b = appendUint16(b, le, uint16(v.Z))
	return b
}

func (v *valueVector3int16) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	v.X = int16(readUint16(&b, le))
	v.Y = int16(readUint16(&b, le))
	v.Z = int16(readUint16(&b, le))
	return n, nil
}

func (v valueVector3int16) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("X: ")
	w.Write(strconv.AppendInt(nil, int64(v.X), 10))

	dumpNewline(w, indent+1)
	w.WriteString("Y: ")
	w.Write(strconv.AppendInt(nil, int64(v.Y), 10))

	dumpNewline(w, indent+1)
	w.WriteString("Z: ")
	w.Write(strconv.AppendInt(nil, int64(v.Z), 10))

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zNumberSequenceKeypoint = zf32 * 3

type valueNumberSequenceKeypoint struct {
	Time, Value, Envelope float32
}

func (v valueNumberSequenceKeypoint) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("Time: ")
	w.Write(strconv.AppendFloat(nil, float64(v.Time), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("Value: ")
	w.Write(strconv.AppendFloat(nil, float64(v.Value), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("Envelope: ")
	w.Write(strconv.AppendFloat(nil, float64(v.Envelope), 'g', -1, 32))

	dumpNewline(w, indent)
	w.WriteByte('}')
}

const zNumberSequence = zArray

type valueNumberSequence []valueNumberSequenceKeypoint

func (valueNumberSequence) Type() typeID {
	return typeNumberSequence
}

func (v valueNumberSequence) BytesLen() int {
	return zArrayLen + zNumberSequenceKeypoint*len(v)
}

func (v valueNumberSequence) Bytes(b []byte) []byte {
	b = appendUint32(b, le, uint32(len(v)))
	for _, nsk := range v {
		b = appendUint32(b, le, math.Float32bits(nsk.Time))
		b = appendUint32(b, le, math.Float32bits(nsk.Value))
		b = appendUint32(b, le, math.Float32bits(nsk.Envelope))
	}
	return b
}

func (v *valueNumberSequence) FromBytes(b []byte) (n int, err error) {
	if b, n, err = checkLengthArray(v, b); err != nil {
		return n, err
	}
	a := make(valueNumberSequence, n)
	for i := 0; i < n; i++ {
		a[i] = valueNumberSequenceKeypoint{
			Time:     math.Float32frombits(readUint32(&b, le)),
			Value:    math.Float32frombits(readUint32(&b, le)),
			Envelope: math.Float32frombits(readUint32(&b, le)),
		}
	}
	*v = a
	return v.BytesLen(), nil
}

func (v valueNumberSequence) Dump(w *bufio.Writer, indent int) {
	w.WriteString("(count:")
	w.Write(strconv.AppendInt(nil, int64(len(v)), 10))
	w.WriteString(") {")
	for i, k := range v {
		dumpNewline(w, indent+1)
		w.Write(strconv.AppendInt(nil, int64(i), 10))
		w.WriteString(": ")
		k.Dump(w, indent+1)
	}
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zColorSequenceKeypoint = zf32 + zColor3 + zf32

type valueColorSequenceKeypoint struct {
	Time     float32
	Value    valueColor3
	Envelope float32
}

func (v valueColorSequenceKeypoint) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("Time: ")
	w.Write(strconv.AppendFloat(nil, float64(v.Time), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("Value: ")
	v.Value.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Envelope: ")
	w.Write(strconv.AppendFloat(nil, float64(v.Envelope), 'g', -1, 32))

	dumpNewline(w, indent)
	w.WriteByte('}')
}

const zColorSequence = zArray

type valueColorSequence []valueColorSequenceKeypoint

func (valueColorSequence) Type() typeID {
	return typeColorSequence
}

func (v valueColorSequence) BytesLen() int {
	return zArrayLen + zColorSequenceKeypoint*len(v)
}

func (v valueColorSequence) Bytes(b []byte) []byte {
	b = appendUint32(b, le, uint32(len(v)))
	for _, csk := range v {
		b = appendUint32(b, le, math.Float32bits(csk.Time))
		b = appendUint32(b, le, math.Float32bits(float32(csk.Value.R)))
		b = appendUint32(b, le, math.Float32bits(float32(csk.Value.G)))
		b = appendUint32(b, le, math.Float32bits(float32(csk.Value.B)))
		b = appendUint32(b, le, math.Float32bits(csk.Envelope))
	}
	return b
}

func (v *valueColorSequence) FromBytes(b []byte) (n int, err error) {
	if b, n, err = checkLengthArray(v, b); err != nil {
		return n, err
	}
	a := make(valueColorSequence, n)
	for i := 0; i < n; i++ {
		a[i] = valueColorSequenceKeypoint{
			Time: math.Float32frombits(readUint32(&b, le)),
			Value: valueColor3{
				R: valueFloat(math.Float32frombits(readUint32(&b, le))),
				G: valueFloat(math.Float32frombits(readUint32(&b, le))),
				B: valueFloat(math.Float32frombits(readUint32(&b, le))),
			},
			Envelope: math.Float32frombits(readUint32(&b, le)),
		}
	}
	*v = a
	return v.BytesLen(), nil
}

func (v valueColorSequence) Dump(w *bufio.Writer, indent int) {
	w.WriteString("(count:")
	w.Write(strconv.AppendInt(nil, int64(len(v)), 10))
	w.WriteString(") {")
	for i, k := range v {
		dumpNewline(w, indent+1)
		w.Write(strconv.AppendInt(nil, int64(i), 10))
		w.WriteString(": ")
		k.Dump(w, indent+1)
	}
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zNumberRange = zf32 * 2

type valueNumberRange struct {
	Min, Max float32
}

func (valueNumberRange) Type() typeID {
	return typeNumberRange
}

func (v valueNumberRange) BytesLen() int {
	return zNumberRange
}

func (v valueNumberRange) Bytes(b []byte) []byte {
	b = appendUint32(b, le, math.Float32bits(v.Min))
	b = appendUint32(b, le, math.Float32bits(v.Max))
	return b
}

func (v *valueNumberRange) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	v.Min = math.Float32frombits(readUint32(&b, le))
	v.Max = math.Float32frombits(readUint32(&b, le))
	return n, nil
}

func (v valueNumberRange) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("Min: ")
	w.Write(strconv.AppendFloat(nil, float64(v.Min), 'g', -1, 32))

	dumpNewline(w, indent+1)
	w.WriteString("Max: ")
	w.Write(strconv.AppendFloat(nil, float64(v.Max), 'g', -1, 32))

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zRect = zVector2 * 2

type valueRect struct {
	Min, Max valueVector2
}

func (valueRect) Type() typeID {
	return typeRect
}

func (v valueRect) BytesLen() int {
	return zRect
}

func (v valueRect) Bytes(b []byte) []byte {
	b = v.Min.Bytes(b)
	b = v.Max.Bytes(b)
	return b
}

func (v *valueRect) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	b = fromBytes(b, &v.Min)
	b = fromBytes(b, &v.Max)
	return n, nil
}

func (v valueRect) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("Min: ")
	v.Min.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Max: ")
	v.Max.Dump(w, indent+1)

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zPhysicalProperties = zCond
const zPhysicalPropertiesCP = zCondLen
const zPhysicalPropertiesFields = zf32 * 5
const zPhysicalPropertiesShort = zPhysicalPropertiesCP
const zPhysicalPropertiesFull = zPhysicalPropertiesCP + zPhysicalPropertiesFields

type valuePhysicalProperties struct {
	CustomPhysics    byte
	Density          float32
	Friction         float32
	Elasticity       float32
	FrictionWeight   float32
	ElasticityWeight float32
}

func (valuePhysicalProperties) Type() typeID {
	return typePhysicalProperties
}

func (v valuePhysicalProperties) BytesLen() int {
	return v.Type().CondSize(v.CustomPhysics)
}

func (v valuePhysicalProperties) ppBytes(b []byte) []byte {
	b = appendUint32(b, le, math.Float32bits(v.Density))
	b = appendUint32(b, le, math.Float32bits(v.Friction))
	b = appendUint32(b, le, math.Float32bits(v.Elasticity))
	b = appendUint32(b, le, math.Float32bits(v.FrictionWeight))
	b = appendUint32(b, le, math.Float32bits(v.ElasticityWeight))
	return b
}

func (v valuePhysicalProperties) Bytes(b []byte) []byte {
	b = append(b, v.CustomPhysics)
	if v.CustomPhysics != 0 {
		b = v.ppBytes(b)
	}
	return b
}

func (v *valuePhysicalProperties) ppFromBytes(b []byte) []byte {
	v.Density = math.Float32frombits(readUint32(&b, le))
	v.Friction = math.Float32frombits(readUint32(&b, le))
	v.Elasticity = math.Float32frombits(readUint32(&b, le))
	v.FrictionWeight = math.Float32frombits(readUint32(&b, le))
	v.ElasticityWeight = math.Float32frombits(readUint32(&b, le))
	return b
}

func (v *valuePhysicalProperties) FromBytes(b []byte) (n int, err error) {
	cond, b, n, err := checkLengthCond(v, b)
	if err != nil {
		return n, err
	}
	v.CustomPhysics = cond
	if cond != 0 {
		b = v.ppFromBytes(b)
	} else {
		v.Density = 0
		v.Friction = 0
		v.Elasticity = 0
		v.FrictionWeight = 0
		v.ElasticityWeight = 0
	}
	return n, nil
}

func (v valuePhysicalProperties) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("CustomPhysics: ")
	w.Write(strconv.AppendUint(nil, uint64(v.CustomPhysics), 10))
	if v.CustomPhysics != 0 {
		dumpNewline(w, indent+1)
		w.WriteString("Density: ")
		w.Write(strconv.AppendFloat(nil, float64(v.Density), 'g', -1, 32))

		dumpNewline(w, indent+1)
		w.WriteString("Friction: ")
		w.Write(strconv.AppendFloat(nil, float64(v.Friction), 'g', -1, 32))

		dumpNewline(w, indent+1)
		w.WriteString("Elasticity: ")
		w.Write(strconv.AppendFloat(nil, float64(v.Elasticity), 'g', -1, 32))

		dumpNewline(w, indent+1)
		w.WriteString("FrictionWeight: ")
		w.Write(strconv.AppendFloat(nil, float64(v.FrictionWeight), 'g', -1, 32))

		dumpNewline(w, indent+1)
		w.WriteString("ElasticityWeight: ")
		w.Write(strconv.AppendFloat(nil, float64(v.ElasticityWeight), 'g', -1, 32))
	}

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zColor3uint8 = zu8 * 3

type valueColor3uint8 struct {
	R, G, B byte
}

func (valueColor3uint8) Type() typeID {
	return typeColor3uint8
}

func (v valueColor3uint8) BytesLen() int {
	return zColor3uint8
}

func (v valueColor3uint8) Bytes(b []byte) []byte {
	return append(b, v.R, v.G, v.B)
}

func (v *valueColor3uint8) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	v.R = b[0]
	v.G = b[1]
	v.B = b[2]
	return n, nil
}

func (v valueColor3uint8) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("R: ")
	w.Write(strconv.AppendUint(nil, uint64(v.R), 10))

	dumpNewline(w, indent+1)
	w.WriteString("G: ")
	w.Write(strconv.AppendUint(nil, uint64(v.G), 10))

	dumpNewline(w, indent+1)
	w.WriteString("B: ")
	w.Write(strconv.AppendUint(nil, uint64(v.B), 10))

	dumpNewline(w, indent)
	w.WriteByte('}')
}

////////////////////////////////////////////////////////////////

const zInt64 = zu64

type valueInt64 int64

func (valueInt64) Type() typeID {
	return typeInt64
}

func (v valueInt64) BytesLen() int {
	return zInt64
}

func (v valueInt64) Bytes(b []byte) []byte {
	return appendUint64(b, be, encodeZigzag64(int64(v)))
}

func (v *valueInt64) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	*v = valueInt64(decodeZigzag64(binary.BigEndian.Uint64(b)))
	return n, nil
}

func (v valueInt64) Dump(w *bufio.Writer, indent int) {
	w.Write(strconv.AppendInt(nil, int64(v), 10))
}

////////////////////////////////////////////////////////////////

const zSharedString = zu32

type valueSharedString uint32

func (valueSharedString) Type() typeID {
	return typeSharedString
}

func (v valueSharedString) BytesLen() int {
	return zSharedString
}

func (v valueSharedString) Bytes(b []byte) []byte {
	return appendUint32(b, be, uint32(v))
}

func (v *valueSharedString) FromBytes(b []byte) (n int, err error) {
	if n, err = checkLengthConst(v, b); err != nil {
		return n, err
	}
	*v = valueSharedString(binary.BigEndian.Uint32(b))
	return n, nil
}

func (v valueSharedString) Dump(w *bufio.Writer, indent int) {
	w.Write(strconv.AppendUint(nil, uint64(v), 10))
}

////////////////////////////////////////////////////////////////

const zOptional = zOpt

type valueOptional struct {
	value
}

func (valueOptional) Type() typeID {
	return typeOptional
}

func (v valueOptional) BytesLen() int {
	if v.value == nil {
		return 0
	}
	return v.value.BytesLen()
}

func (v valueOptional) Bytes(b []byte) []byte {
	// Unused.

	if v.value == nil {
		return append(b, 0)
	}
	b = append(b, byte(v.value.Type()))
	b = v.value.Bytes(b)
	return b
}

func (v *valueOptional) FromBytes(b []byte) (n int, err error) {
	// Unused.

	if len(b) < zb {
		return 0, buflenError{exp: zb, got: len(b)}
	}
	t := typeID(b[0])
	if t == 0 {
		v.value = nil
		return
	}
	b = b[zb:]
	value := newValue(t)
	nn, err := value.FromBytes(b)
	if err != nil {
		return n, err
	}
	n += nn
	v.value = value
	return n, nil
}

func (v valueOptional) Dump(w *bufio.Writer, indent int) {
	if v.value == nil {
		w.WriteString("nil")
		return
	}
	v.value.Dump(w, indent)
}

////////////////////////////////////////////////////////////////
