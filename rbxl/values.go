package rbxl

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	"github.com/robloxapi/rbxfile"
)

// Expected maximum length of fielder.fieldLen() slice. This MUST be set to the
// maximum of all implementations.
const maxFieldLen = 4

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

	// Number of bytes used to contain length of array-like type.
	zArrayLen = 4

	// Variable size.
	zVar = -1

	// Invalid size.
	zInvalid = 0
)

// typeID represents a type that can be serialized.
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
)

// Valid returns whether the type has a valid value.
func (t typeID) Valid() bool {
	return typeString <= t && t <= typeSharedString
}

// Size returns the number of bytes required to hold a value of the type.
// Returns < 0 if the size depends on the value, and 0 if the type is invalid.
//
// A Size() of < 0 with a non-zero FieldSize() indicates an array-like type,
// where the first 4 bytes are the size of the array, and each element has a
// size of FieldSize().
//
// A Size() of < 0 with a FieldSize() of 0 indicates a type with a customized
// size.
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
	default:
		return zInvalid
	}
}

// FieldSize returns the number of bytes of each field within a value of the
// type, where the type is a variable-length array of fields. Returns 0 if the
// type is invalid or not array-like.
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

	// Bytes encodes value to buf, panicking if buf is shorter than BytesLen().
	Bytes(buf []byte)

	// FromBytes decodes the value from buf. Returns an error if the value could
	// not be decoded. If successful, BytesLen() will return the number of bytes
	// read from buf.
	FromBytes(buf []byte) error

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
	typ typeID
	exp uint64
	got int
}

func (err buflenError) Error() string {
	return fmt.Sprintf("%s: expected %d bytes, got %d", err.typ, err.exp, err.got)
}

// checklen does a basic check of the buffer's length against a value with a
// constant expected byte length.
func checklen(v value, b []byte) error {
	if len(b) < v.BytesLen() {
		return buflenError{typ: v.Type(), exp: uint64(v.BytesLen()), got: len(b)}
	}
	return nil
}

// checkvarlen checks the buffer's length to make sure it can be decoded into
// the value. The first 4 bytes are decoded as the number of fields of the
// value, and the remaining length of the buffer is expected to be
// v.Type().FieldSize()*length. Returns the remaining buffer and the number of
// fields. Returns an error if the buffer is too short.
func checkvarlen(v value, b []byte) ([]byte, int, error) {
	if len(b) < zArrayLen {
		return b, 0, buflenError{typ: v.Type(), exp: zArrayLen, got: len(b)}
	}
	length := binary.LittleEndian.Uint32(b[:zArrayLen])
	if n := zArrayLen + uint64(v.Type().FieldSize())*uint64(length); uint64(len(b)) < n {
		return b, 0, buflenError{typ: v.Type(), exp: n, got: len(b)}
	}
	return b[zArrayLen:], int(length), nil
}

////////////////////////////////////////////////////////////////

const zString = zVar

type valueString []byte

func (valueString) Type() typeID {
	return typeString
}

func (v valueString) BytesLen() int {
	return zArrayLen + len(v)
}

func (v valueString) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint32(b, uint32(len(v)))
	copy(b[zArrayLen:], v)
}

func (v *valueString) FromBytes(b []byte) error {
	b, n, err := checkvarlen(v, b)
	if err != nil {
		return err
	}
	*v = make(valueString, n)
	copy(*v, b)
	return nil
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

func (v valueBool) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	if v {
		b[0] = 1
	} else {
		b[0] = 0
	}
}

func (v *valueBool) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = b[0] != 0
	return nil
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

func (v valueInt) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, encodeZigzag32(int32(v)))
}

func (v *valueInt) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = valueInt(decodeZigzag32(binary.BigEndian.Uint32(b)))
	return nil
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

func (v valueFloat) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, encodeRobloxFloat(float32(v)))
}

func (v *valueFloat) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = valueFloat(decodeRobloxFloat(binary.BigEndian.Uint32(b)))
	return nil
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

func (v valueDouble) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint64(b, math.Float64bits(float64(v)))
}

func (v *valueDouble) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = valueDouble(math.Float64frombits(binary.LittleEndian.Uint64(b)))
	return nil
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

func (v valueUDim) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.Scale.Bytes(b[0:4])
	v.Offset.Bytes(b[4:8])
}

func (v *valueUDim) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.Scale.FromBytes(b[0:4])
	v.Offset.FromBytes(b[4:8])
	return nil
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

func (valueUDim) fieldLen() []int {
	return []int{4, 4}
}

func (v *valueUDim) fieldSet(i int, b []byte) (err error) {
	switch i {
	case 0:
		err = v.Scale.FromBytes(b)
	case 1:
		err = v.Offset.FromBytes(b)
	}
	return
}

func (v valueUDim) fieldGet(i int, b []byte) {
	switch i {
	case 0:
		v.Scale.Bytes(b)
	case 1:
		v.Offset.Bytes(b)
	}
	return
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

func (v valueUDim2) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.ScaleX.Bytes(b[0:4])
	v.ScaleY.Bytes(b[4:8])
	v.OffsetX.Bytes(b[8:12])
	v.OffsetY.Bytes(b[12:16])
}

func (v *valueUDim2) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.ScaleX.FromBytes(b[0:4])
	v.ScaleY.FromBytes(b[4:8])
	v.OffsetX.FromBytes(b[8:12])
	v.OffsetY.FromBytes(b[12:16])
	return nil
}

func (v valueUDim2) Dump(w *bufio.Writer, indent int) {
	w.WriteByte('{')

	dumpNewline(w, indent+1)
	w.WriteString("Scale.X: ")
	v.ScaleX.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Scale.Y: ")
	v.OffsetX.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Offset.X: ")
	v.ScaleY.Dump(w, indent+1)

	dumpNewline(w, indent+1)
	w.WriteString("Offset.Y: ")
	v.OffsetY.Dump(w, indent+1)

	dumpNewline(w, indent)
	w.WriteByte('}')
}

func (valueUDim2) fieldLen() []int {
	return []int{4, 4, 4, 4}
}

func (v *valueUDim2) fieldSet(i int, b []byte) (err error) {
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

func (v valueUDim2) fieldGet(i int, b []byte) {
	switch i {
	case 0:
		v.ScaleX.Bytes(b)
	case 1:
		v.ScaleY.Bytes(b)
	case 2:
		v.OffsetX.Bytes(b)
	case 3:
		v.OffsetY.Bytes(b)
	}
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

func (v valueRay) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(v.OriginX))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(v.OriginY))
	binary.LittleEndian.PutUint32(b[8:12], math.Float32bits(v.OriginZ))
	binary.LittleEndian.PutUint32(b[12:16], math.Float32bits(v.DirectionX))
	binary.LittleEndian.PutUint32(b[16:20], math.Float32bits(v.DirectionY))
	binary.LittleEndian.PutUint32(b[20:24], math.Float32bits(v.DirectionZ))
}

func (v *valueRay) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.OriginX = math.Float32frombits(binary.LittleEndian.Uint32(b[0:4]))
	v.OriginY = math.Float32frombits(binary.LittleEndian.Uint32(b[4:8]))
	v.OriginZ = math.Float32frombits(binary.LittleEndian.Uint32(b[8:12]))
	v.DirectionX = math.Float32frombits(binary.LittleEndian.Uint32(b[12:16]))
	v.DirectionY = math.Float32frombits(binary.LittleEndian.Uint32(b[16:20]))
	v.DirectionZ = math.Float32frombits(binary.LittleEndian.Uint32(b[20:24]))
	return nil
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

func (v valueFaces) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	flags := [6]bool{v.Right, v.Top, v.Back, v.Left, v.Bottom, v.Front}
	b[0] = 0
	for i, flag := range flags {
		if flag {
			b[0] |= 1 << uint(i)
		}
	}
}

func (v *valueFaces) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.Right = b[0]&(1<<0) != 0
	v.Top = b[0]&(1<<1) != 0
	v.Back = b[0]&(1<<2) != 0
	v.Left = b[0]&(1<<3) != 0
	v.Bottom = b[0]&(1<<4) != 0
	v.Front = b[0]&(1<<5) != 0
	return nil
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

func (v valueAxes) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	flags := [3]bool{v.X, v.Y, v.Z}
	b[0] = 0
	for i, flag := range flags {
		if flag {
			b[0] |= 1 << uint(i)
		}
	}
}

func (v *valueAxes) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.X = b[0]&(1<<0) != 0
	v.Y = b[0]&(1<<1) != 0
	v.Z = b[0]&(1<<2) != 0
	return nil
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

func (v valueBrickColor) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, uint32(v))
}

func (v *valueBrickColor) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = valueBrickColor(binary.BigEndian.Uint32(b))
	return nil
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

func (v valueColor3) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.R.Bytes(b[0:4])
	v.G.Bytes(b[4:8])
	v.B.Bytes(b[8:12])
}

func (v *valueColor3) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.R.FromBytes(b[0:4])
	v.G.FromBytes(b[4:8])
	v.B.FromBytes(b[8:12])
	return nil
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

func (valueColor3) fieldLen() []int {
	return []int{4, 4, 4}
}

func (v *valueColor3) fieldSet(i int, b []byte) (err error) {
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

func (v valueColor3) fieldGet(i int, b []byte) {
	switch i {
	case 0:
		v.R.Bytes(b)
	case 1:
		v.G.Bytes(b)
	case 2:
		v.B.Bytes(b)
	}
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

func (v valueVector2) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.X.Bytes(b[0:4])
	v.Y.Bytes(b[4:8])
}

func (v *valueVector2) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.X.FromBytes(b[0:4])
	v.Y.FromBytes(b[4:8])
	return nil
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

func (valueVector2) fieldLen() []int {
	return []int{4, 4}
}

func (v *valueVector2) fieldSet(i int, b []byte) (err error) {
	switch i {
	case 0:
		err = v.X.FromBytes(b)
	case 1:
		err = v.Y.FromBytes(b)
	}
	return
}

func (v valueVector2) fieldGet(i int, b []byte) {
	switch i {
	case 0:
		v.X.Bytes(b)
	case 1:
		v.Y.Bytes(b)
	}
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

func (v valueVector3) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.X.Bytes(b[0:4])
	v.Y.Bytes(b[4:8])
	v.Z.Bytes(b[8:12])
}

func (v *valueVector3) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.X.FromBytes(b[0:4])
	v.Y.FromBytes(b[4:8])
	v.Z.FromBytes(b[8:12])
	return nil
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

func (valueVector3) fieldLen() []int {
	return []int{4, 4, 4}
}

func (v *valueVector3) fieldSet(i int, b []byte) (err error) {
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

func (v valueVector3) fieldGet(i int, b []byte) {
	switch i {
	case 0:
		v.X.Bytes(b)
	case 1:
		v.Y.Bytes(b)
	case 2:
		v.Z.Bytes(b)
	}
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

func (v valueVector2int16) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint16(b[0:2], uint16(v.X))
	binary.LittleEndian.PutUint16(b[2:4], uint16(v.Y))
}

func (v *valueVector2int16) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.X = int16(binary.LittleEndian.Uint16(b[0:2]))
	v.Y = int16(binary.LittleEndian.Uint16(b[2:4]))
	return nil
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

const zCFrame = zVar
const zCFrameSp = zu8
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
	if v.Special == 0 {
		return zCFrameFull
	}
	return zCFrameShort
}

func (v valueCFrame) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	n := 1
	if v.Special == 0 {
		b[0] = 0
		r := b[zCFrameSp:]
		for i, f := range v.Rotation {
			binary.LittleEndian.PutUint32(r[i*zf32:i*zf32+zf32], math.Float32bits(f))
		}
		n += len(v.Rotation) * zf32
	} else {
		b[0] = v.Special
	}
	v.Position.Bytes(b[n:])
}

func (v *valueCFrame) FromBytes(b []byte) error {
	if len(b) < zCFrameSp {
		return buflenError{typ: v.Type(), exp: zCFrameSp, got: len(b)}
	}
	if b[0] == 0 && len(b) < zCFrameFull {
		return buflenError{typ: v.Type(), exp: zCFrameFull, got: len(b)}
	} else if b[0] != 0 && len(b) < zCFrameShort {
		return buflenError{typ: v.Type(), exp: zCFrameShort, got: len(b)}
	}
	v.Special = b[0]
	if b[0] == 0 {
		r := b[zCFrameSp:]
		for i := range v.Rotation {
			v.Rotation[i] = math.Float32frombits(binary.LittleEndian.Uint32(r[i*zf32 : i*zf32+zf32]))
		}
		v.Position.FromBytes(b[zCFrameSp+zCFrameRo:])
	} else {
		for i := range v.Rotation {
			v.Rotation[i] = 0
		}
		v.Position.FromBytes(b[zCFrameSp:])
	}
	return nil
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

const zCFrameQuat = -1
const zCFrameQuatSp = zu8
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
	if v.Special == 0 {
		return zCFrameQuatFull
	}
	return zCFrameQuatShort
}

func (v valueCFrameQuat) quatBytes(b []byte) {
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(v.QX))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(v.QY))
	binary.LittleEndian.PutUint32(b[8:12], math.Float32bits(v.QZ))
	binary.LittleEndian.PutUint32(b[12:16], math.Float32bits(v.QW))
}

func (v valueCFrameQuat) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	n := 1
	if v.Special == 0 {
		b[0] = 0
		v.quatBytes(b[zCFrameQuatSp:])
		n += zCFrameQuatQ
	} else {
		b[0] = v.Special
	}
	v.Position.Bytes(b[n:])
}

func (v *valueCFrameQuat) quatFromBytes(b []byte) {
	v.QX = math.Float32frombits(binary.LittleEndian.Uint32(b[0:4]))
	v.QY = math.Float32frombits(binary.LittleEndian.Uint32(b[4:8]))
	v.QZ = math.Float32frombits(binary.LittleEndian.Uint32(b[8:12]))
	v.QW = math.Float32frombits(binary.LittleEndian.Uint32(b[12:16]))
}

func (v *valueCFrameQuat) FromBytes(b []byte) error {
	if len(b) < zCFrameQuatSp {
		return buflenError{typ: v.Type(), exp: zCFrameQuatSp, got: len(b)}
	}
	if b[0] == 0 && len(b) < zCFrameQuatFull {
		return buflenError{typ: v.Type(), exp: zCFrameQuatFull, got: len(b)}
	} else if b[0] != 0 && len(b) < zCFrameQuatShort {
		return buflenError{typ: v.Type(), exp: zCFrameQuatShort, got: len(b)}
	}
	v.Special = b[0]
	if b[0] == 0 {
		v.quatFromBytes(b[zCFrameQuatSp:])
		v.Position.FromBytes(b[zCFrameQuatSp+zCFrameQuatQ:])
	} else {
		v.QX = 0
		v.QY = 0
		v.QZ = 0
		v.QW = 0
		v.Position.FromBytes(b[zCFrameQuatSp:])
	}
	return nil
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

func (v valueToken) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, uint32(v))
}

func (v *valueToken) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = valueToken(binary.BigEndian.Uint32(b))
	return nil
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

func (v valueReference) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, encodeZigzag32(int32(v)))
}

func (v *valueReference) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = valueReference(decodeZigzag32(binary.BigEndian.Uint32(b)))
	return nil
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

func (v valueVector3int16) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint16(b[0:2], uint16(v.X))
	binary.LittleEndian.PutUint16(b[2:4], uint16(v.Y))
	binary.LittleEndian.PutUint16(b[4:6], uint16(v.Z))
}

func (v *valueVector3int16) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.X = int16(binary.LittleEndian.Uint16(b[0:2]))
	v.Y = int16(binary.LittleEndian.Uint16(b[2:4]))
	v.Z = int16(binary.LittleEndian.Uint16(b[4:6]))
	return nil
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

const zNumberSequence = zVar

type valueNumberSequence []valueNumberSequenceKeypoint

func (valueNumberSequence) Type() typeID {
	return typeNumberSequence
}

func (v valueNumberSequence) BytesLen() int {
	return zArrayLen + zNumberSequenceKeypoint*len(v)
}

func (v valueNumberSequence) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint32(b, uint32(len(v)))
	ba := b[zArrayLen:]
	for i, nsk := range v {
		bk := ba[i*zNumberSequenceKeypoint:]
		binary.LittleEndian.PutUint32(bk[0:4], math.Float32bits(nsk.Time))
		binary.LittleEndian.PutUint32(bk[4:8], math.Float32bits(nsk.Value))
		binary.LittleEndian.PutUint32(bk[8:12], math.Float32bits(nsk.Envelope))
	}
}

func (v *valueNumberSequence) FromBytes(b []byte) error {
	b, n, err := checkvarlen(v, b)
	if err != nil {
		return err
	}
	a := make(valueNumberSequence, n)
	for i := 0; i < n; i++ {
		bk := b[i*zNumberSequenceKeypoint:]
		a[i] = valueNumberSequenceKeypoint{
			Time:     math.Float32frombits(binary.LittleEndian.Uint32(bk[0:4])),
			Value:    math.Float32frombits(binary.LittleEndian.Uint32(bk[4:8])),
			Envelope: math.Float32frombits(binary.LittleEndian.Uint32(bk[8:12])),
		}
	}
	*v = a
	return nil
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

const zColorSequence = zVar

type valueColorSequence []valueColorSequenceKeypoint

func (valueColorSequence) Type() typeID {
	return typeColorSequence
}

func (v valueColorSequence) BytesLen() int {
	return zArrayLen + zColorSequenceKeypoint*len(v)
}

func (v valueColorSequence) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint32(b, uint32(len(v)))
	ba := b[zArrayLen:]
	for i, csk := range v {
		bk := ba[i*zColorSequenceKeypoint:]
		binary.LittleEndian.PutUint32(bk[0:4], math.Float32bits(csk.Time))
		binary.LittleEndian.PutUint32(bk[4:8], math.Float32bits(float32(csk.Value.R)))
		binary.LittleEndian.PutUint32(bk[8:12], math.Float32bits(float32(csk.Value.G)))
		binary.LittleEndian.PutUint32(bk[12:16], math.Float32bits(float32(csk.Value.B)))
		binary.LittleEndian.PutUint32(bk[16:20], math.Float32bits(csk.Envelope))
	}
}

func (v *valueColorSequence) FromBytes(b []byte) error {
	b, n, err := checkvarlen(v, b)
	if err != nil {
		return err
	}
	a := make(valueColorSequence, n)
	for i := 0; i < n; i++ {
		bk := b[i*zColorSequenceKeypoint:]
		c3 := *new(valueColor3)
		c3.FromBytes(bk[4:16])
		a[i] = valueColorSequenceKeypoint{
			Time: math.Float32frombits(binary.LittleEndian.Uint32(bk[0:4])),
			Value: valueColor3{
				R: valueFloat(math.Float32frombits(binary.LittleEndian.Uint32(bk[4:8]))),
				G: valueFloat(math.Float32frombits(binary.LittleEndian.Uint32(bk[8:12]))),
				B: valueFloat(math.Float32frombits(binary.LittleEndian.Uint32(bk[12:16]))),
			},
			Envelope: math.Float32frombits(binary.LittleEndian.Uint32(bk[16:20])),
		}
	}
	*v = a
	return nil
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

func (v valueNumberRange) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(v.Min))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(v.Max))
}

func (v *valueNumberRange) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.Min = math.Float32frombits(binary.LittleEndian.Uint32(b[0:4]))
	v.Max = math.Float32frombits(binary.LittleEndian.Uint32(b[4:8]))
	return nil
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

func (v valueRect) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.Min.Bytes(b[0:8])
	v.Max.Bytes(b[8:16])
}

func (v *valueRect) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.Min.FromBytes(b[0:8])
	v.Max.FromBytes(b[8:16])
	return nil
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

func (valueRect) fieldLen() []int {
	return []int{4, 4, 4, 4}
}

func (v *valueRect) fieldSet(i int, b []byte) (err error) {
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

func (v valueRect) fieldGet(i int, b []byte) {
	switch i {
	case 0:
		v.Min.X.Bytes(b)
	case 1:
		v.Min.Y.Bytes(b)
	case 2:
		v.Max.X.Bytes(b)
	case 3:
		v.Max.Y.Bytes(b)
	}
}

////////////////////////////////////////////////////////////////

const zPhysicalProperties = zVar
const zPhysicalPropertiesCP = zb
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
	if v.CustomPhysics == 0 {
		return zPhysicalPropertiesShort
	}
	return zPhysicalPropertiesFull
}

func (v valuePhysicalProperties) ppBytes(b []byte) {
	binary.LittleEndian.PutUint32(b[0*zf32:0*zf32+zf32], math.Float32bits(v.Density))
	binary.LittleEndian.PutUint32(b[1*zf32:1*zf32+zf32], math.Float32bits(v.Friction))
	binary.LittleEndian.PutUint32(b[2*zf32:2*zf32+zf32], math.Float32bits(v.Elasticity))
	binary.LittleEndian.PutUint32(b[3*zf32:3*zf32+zf32], math.Float32bits(v.FrictionWeight))
	binary.LittleEndian.PutUint32(b[4*zf32:4*zf32+zf32], math.Float32bits(v.ElasticityWeight))
}

func (v valuePhysicalProperties) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	b[0] = v.CustomPhysics
	if v.CustomPhysics != 0 {
		v.ppBytes(b[zPhysicalPropertiesCP:])
	}
}

func (v *valuePhysicalProperties) ppFromBytes(b []byte) {
	v.Density = math.Float32frombits(binary.LittleEndian.Uint32(b[0*zf32 : 0*zf32+zf32]))
	v.Friction = math.Float32frombits(binary.LittleEndian.Uint32(b[1*zf32 : 1*zf32+zf32]))
	v.Elasticity = math.Float32frombits(binary.LittleEndian.Uint32(b[2*zf32 : 2*zf32+zf32]))
	v.FrictionWeight = math.Float32frombits(binary.LittleEndian.Uint32(b[3*zf32 : 3*zf32+zf32]))
	v.ElasticityWeight = math.Float32frombits(binary.LittleEndian.Uint32(b[4*zf32 : 4*zf32+zf32]))
}

func (v *valuePhysicalProperties) FromBytes(b []byte) error {
	if len(b) < zPhysicalPropertiesCP {
		return buflenError{typ: v.Type(), exp: zPhysicalPropertiesCP, got: len(b)}
	}
	if b[0] != 0 && len(b) < zPhysicalPropertiesFull {
		return buflenError{typ: v.Type(), exp: zPhysicalPropertiesFull, got: len(b)}
	}
	v.CustomPhysics = b[0]
	if v.CustomPhysics != 0 {
		v.ppFromBytes(b[zPhysicalPropertiesCP:])
	} else {
		v.Density = 0
		v.Friction = 0
		v.Elasticity = 0
		v.FrictionWeight = 0
		v.ElasticityWeight = 0
	}
	return nil
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

const zColor3uint8 = zb * 3

type valueColor3uint8 struct {
	R, G, B byte
}

func (valueColor3uint8) Type() typeID {
	return typeColor3uint8
}

func (v valueColor3uint8) BytesLen() int {
	return zColor3uint8
}

func (v valueColor3uint8) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	b[0] = v.R
	b[1] = v.G
	b[2] = v.B
}

func (v *valueColor3uint8) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.R = b[0]
	v.G = b[1]
	v.B = b[2]
	return nil
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

func (valueColor3uint8) fieldLen() []int {
	return []int{1, 1, 1}
}

func (v *valueColor3uint8) fieldSet(i int, b []byte) (err error) {
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

func (v valueColor3uint8) fieldGet(i int, b []byte) {
	switch i {
	case 0:
		b[0] = v.R
	case 1:
		b[0] = v.G
	case 2:
		b[0] = v.B
	}
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

func (v valueInt64) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint64(b, encodeZigzag64(int64(v)))
}

func (v *valueInt64) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = valueInt64(decodeZigzag64(binary.BigEndian.Uint64(b)))
	return nil
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

func (v valueSharedString) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, uint32(v))
}

func (v *valueSharedString) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = valueSharedString(binary.BigEndian.Uint32(b))
	return nil
}

func (v valueSharedString) Dump(w *bufio.Writer, indent int) {
	w.Write(strconv.AppendUint(nil, uint64(v), 10))
}

////////////////////////////////////////////////////////////////
