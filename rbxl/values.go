package rbxl

import (
	"encoding/binary"
	"fmt"
	"math"

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

// FromValueType returns the Type corresponding to a given rbxfile.Type.
func FromValueType(t rbxfile.Type) typeID {
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

// Value represents a value of a certain Type.
type Value interface {
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
}

// NewValue returns new Value of the given Type. The initial value will not
// necessarily be the zero for the type. If the given type is invalid, then a
// nil value is returned.
func NewValue(typ typeID) Value {
	switch typ {
	case typeString:
		return new(ValueString)
	case typeBool:
		return new(ValueBool)
	case typeInt:
		return new(ValueInt)
	case typeFloat:
		return new(ValueFloat)
	case typeDouble:
		return new(ValueDouble)
	case typeUDim:
		return new(ValueUDim)
	case typeUDim2:
		return new(ValueUDim2)
	case typeRay:
		return new(ValueRay)
	case typeFaces:
		return new(ValueFaces)
	case typeAxes:
		return new(ValueAxes)
	case typeBrickColor:
		return new(ValueBrickColor)
	case typeColor3:
		return new(ValueColor3)
	case typeVector2:
		return new(ValueVector2)
	case typeVector3:
		return new(ValueVector3)
	case typeVector2int16:
		return new(ValueVector2int16)
	case typeCFrame:
		return new(ValueCFrame)
	case typeCFrameQuat:
		return new(ValueCFrameQuat)
	case typeToken:
		return new(ValueToken)
	case typeReference:
		return new(ValueReference)
	case typeVector3int16:
		return new(ValueVector3int16)
	case typeNumberSequence:
		return new(ValueNumberSequence)
	case typeColorSequence:
		return new(ValueColorSequence)
	case typeNumberRange:
		return new(ValueNumberRange)
	case typeRect:
		return new(ValueRect)
	case typePhysicalProperties:
		return new(ValuePhysicalProperties)
	case typeColor3uint8:
		return new(ValueColor3uint8)
	case typeInt64:
		return new(ValueInt64)
	case typeSharedString:
		return new(ValueSharedString)
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
func checklen(v Value, b []byte) error {
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
func checkvarlen(v Value, b []byte) ([]byte, int, error) {
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

type ValueString []byte

func (ValueString) Type() typeID {
	return typeString
}

func (v ValueString) BytesLen() int {
	return zArrayLen + len(v)
}

func (v ValueString) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint32(b, uint32(len(v)))
	copy(b[zArrayLen:], v)
}

func (v *ValueString) FromBytes(b []byte) error {
	b, n, err := checkvarlen(v, b)
	if err != nil {
		return err
	}
	*v = make(ValueString, n)
	copy(*v, b)
	return nil
}

////////////////////////////////////////////////////////////////

const zBool = zb

type ValueBool bool

func (ValueBool) Type() typeID {
	return typeBool
}

func (v ValueBool) BytesLen() int {
	return zBool
}

func (v ValueBool) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	if v {
		b[0] = 1
	} else {
		b[0] = 0
	}
}

func (v *ValueBool) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = b[0] != 0
	return nil
}

////////////////////////////////////////////////////////////////

const zInt = zi32

type ValueInt int32

func (ValueInt) Type() typeID {
	return typeInt
}

func (v ValueInt) BytesLen() int {
	return zInt
}

func (v ValueInt) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, encodeZigzag32(int32(v)))
}

func (v *ValueInt) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = ValueInt(decodeZigzag32(binary.BigEndian.Uint32(b)))
	return nil
}

////////////////////////////////////////////////////////////////

const zFloat = zf32

type ValueFloat float32

func (ValueFloat) Type() typeID {
	return typeFloat
}

func (v ValueFloat) BytesLen() int {
	return zFloat
}

func (v ValueFloat) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, encodeRobloxFloat(float32(v)))
}

func (v *ValueFloat) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = ValueFloat(decodeRobloxFloat(binary.BigEndian.Uint32(b)))
	return nil
}

////////////////////////////////////////////////////////////////

const zDouble = zf64

type ValueDouble float64

func (ValueDouble) Type() typeID {
	return typeDouble
}

func (v ValueDouble) BytesLen() int {
	return zDouble
}

func (v ValueDouble) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint64(b, math.Float64bits(float64(v)))
}

func (v *ValueDouble) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = ValueDouble(math.Float64frombits(binary.LittleEndian.Uint64(b)))
	return nil
}

////////////////////////////////////////////////////////////////

const zUDim = zFloat + zInt

type ValueUDim struct {
	Scale  ValueFloat
	Offset ValueInt
}

func (ValueUDim) Type() typeID {
	return typeUDim
}

func (v ValueUDim) BytesLen() int {
	return zUDim
}

func (v ValueUDim) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.Scale.Bytes(b[0:4])
	v.Offset.Bytes(b[4:8])
}

func (v *ValueUDim) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.Scale.FromBytes(b[0:4])
	v.Offset.FromBytes(b[4:8])
	return nil
}

func (ValueUDim) fieldLen() []int {
	return []int{4, 4}
}

func (v *ValueUDim) fieldSet(i int, b []byte) (err error) {
	switch i {
	case 0:
		err = v.Scale.FromBytes(b)
	case 1:
		err = v.Offset.FromBytes(b)
	}
	return
}

func (v ValueUDim) fieldGet(i int, b []byte) {
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

type ValueUDim2 struct {
	ScaleX  ValueFloat
	ScaleY  ValueFloat
	OffsetX ValueInt
	OffsetY ValueInt
}

func (ValueUDim2) Type() typeID {
	return typeUDim2
}

func (v ValueUDim2) BytesLen() int {
	return zUDim2
}

func (v ValueUDim2) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.ScaleX.Bytes(b[0:4])
	v.ScaleY.Bytes(b[4:8])
	v.OffsetX.Bytes(b[8:12])
	v.OffsetY.Bytes(b[12:16])
}

func (v *ValueUDim2) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
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

func (v ValueUDim2) fieldGet(i int, b []byte) {
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

type ValueRay struct {
	OriginX    float32
	OriginY    float32
	OriginZ    float32
	DirectionX float32
	DirectionY float32
	DirectionZ float32
}

func (ValueRay) Type() typeID {
	return typeRay
}

func (v ValueRay) BytesLen() int {
	return zRay
}

func (v ValueRay) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(v.OriginX))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(v.OriginY))
	binary.LittleEndian.PutUint32(b[8:12], math.Float32bits(v.OriginZ))
	binary.LittleEndian.PutUint32(b[12:16], math.Float32bits(v.DirectionX))
	binary.LittleEndian.PutUint32(b[16:20], math.Float32bits(v.DirectionY))
	binary.LittleEndian.PutUint32(b[20:24], math.Float32bits(v.DirectionZ))
}

func (v *ValueRay) FromBytes(b []byte) error {
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

////////////////////////////////////////////////////////////////

const zFaces = zu8

type ValueFaces struct {
	Right, Top, Back, Left, Bottom, Front bool
}

func (ValueFaces) Type() typeID {
	return typeFaces
}

func (v ValueFaces) BytesLen() int {
	return zFaces
}

func (v ValueFaces) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	flags := [6]bool{v.Right, v.Top, v.Back, v.Left, v.Bottom, v.Front}
	b[0] = 0
	for i, flag := range flags {
		if flag {
			b[0] |= 1 << uint(i)
		}
	}
}

func (v *ValueFaces) FromBytes(b []byte) error {
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

////////////////////////////////////////////////////////////////

const zAxes = zu8

type ValueAxes struct {
	X, Y, Z bool
}

func (ValueAxes) Type() typeID {
	return typeAxes
}

func (v ValueAxes) BytesLen() int {
	return zAxes
}

func (v ValueAxes) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	flags := [3]bool{v.X, v.Y, v.Z}
	b[0] = 0
	for i, flag := range flags {
		if flag {
			b[0] |= 1 << uint(i)
		}
	}
}

func (v *ValueAxes) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.X = b[0]&(1<<0) != 0
	v.Y = b[0]&(1<<1) != 0
	v.Z = b[0]&(1<<2) != 0
	return nil
}

////////////////////////////////////////////////////////////////

const zBrickColor = zu32

type ValueBrickColor uint32

func (ValueBrickColor) Type() typeID {
	return typeBrickColor
}

func (v ValueBrickColor) BytesLen() int {
	return zBrickColor
}

func (v ValueBrickColor) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, uint32(v))
}

func (v *ValueBrickColor) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = ValueBrickColor(binary.BigEndian.Uint32(b))
	return nil
}

////////////////////////////////////////////////////////////////

const zColor3 = zFloat * 3

type ValueColor3 struct {
	R, G, B ValueFloat
}

func (ValueColor3) Type() typeID {
	return typeColor3
}

func (v ValueColor3) BytesLen() int {
	return zColor3
}

func (v ValueColor3) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.R.Bytes(b[0:4])
	v.G.Bytes(b[4:8])
	v.B.Bytes(b[8:12])
}

func (v *ValueColor3) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
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

func (v ValueColor3) fieldGet(i int, b []byte) {
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

type ValueVector2 struct {
	X, Y ValueFloat
}

func (ValueVector2) Type() typeID {
	return typeVector2
}

func (v ValueVector2) BytesLen() int {
	return zVector2
}

func (v ValueVector2) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.X.Bytes(b[0:4])
	v.Y.Bytes(b[4:8])
}

func (v *ValueVector2) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
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

func (v ValueVector2) fieldGet(i int, b []byte) {
	switch i {
	case 0:
		v.X.Bytes(b)
	case 1:
		v.Y.Bytes(b)
	}
}

////////////////////////////////////////////////////////////////

const zVector3 = zFloat * 3

type ValueVector3 struct {
	X, Y, Z ValueFloat
}

func (ValueVector3) Type() typeID {
	return typeVector3
}

func (v ValueVector3) BytesLen() int {
	return zVector3
}

func (v ValueVector3) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.X.Bytes(b[0:4])
	v.Y.Bytes(b[4:8])
	v.Z.Bytes(b[8:12])
}

func (v *ValueVector3) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
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

func (v ValueVector3) fieldGet(i int, b []byte) {
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

type ValueVector2int16 struct {
	X, Y int16
}

func (ValueVector2int16) Type() typeID {
	return typeVector2int16
}

func (v ValueVector2int16) BytesLen() int {
	return zVector2int16
}

func (v ValueVector2int16) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint16(b[0:2], uint16(v.X))
	binary.LittleEndian.PutUint16(b[2:4], uint16(v.Y))
}

func (v *ValueVector2int16) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.X = int16(binary.LittleEndian.Uint16(b[0:2]))
	v.Y = int16(binary.LittleEndian.Uint16(b[2:4]))
	return nil
}

////////////////////////////////////////////////////////////////

const zCFrame = zVar
const zCFrameSp = zu8
const zCFrameRo = zf32 * 9
const zCFrameFull = zCFrameSp + zCFrameRo + zVector3
const zCFrameShort = zCFrameSp + zVector3

type ValueCFrame struct {
	Special  uint8
	Rotation [9]float32
	Position ValueVector3
}

func (ValueCFrame) Type() typeID {
	return typeCFrame
}

func (v ValueCFrame) BytesLen() int {
	if v.Special == 0 {
		return zCFrameFull
	}
	return zCFrameShort
}

func (v ValueCFrame) Bytes(b []byte) {
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

func (v *ValueCFrame) FromBytes(b []byte) error {
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

func sqrt32(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

// ToCFrameQuat converts the value to a ValueCFrameQuat.
func (v ValueCFrame) ToCFrameQuat() (q ValueCFrameQuat) {
	if v.Special != 0 {
		return ValueCFrameQuat{
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

type ValueCFrameQuat struct {
	Special        uint8
	QX, QY, QZ, QW float32
	Position       ValueVector3
}

func (ValueCFrameQuat) Type() typeID {
	return typeCFrameQuat
}

func (v ValueCFrameQuat) BytesLen() int {
	if v.Special == 0 {
		return zCFrameQuatFull
	}
	return zCFrameQuatShort
}

func (v ValueCFrameQuat) quatBytes(b []byte) {
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(v.QX))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(v.QY))
	binary.LittleEndian.PutUint32(b[8:12], math.Float32bits(v.QZ))
	binary.LittleEndian.PutUint32(b[12:16], math.Float32bits(v.QW))
}

func (v ValueCFrameQuat) Bytes(b []byte) {
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

func (v *ValueCFrameQuat) quatFromBytes(b []byte) {
	v.QX = math.Float32frombits(binary.LittleEndian.Uint32(b[0:4]))
	v.QY = math.Float32frombits(binary.LittleEndian.Uint32(b[4:8]))
	v.QZ = math.Float32frombits(binary.LittleEndian.Uint32(b[8:12]))
	v.QW = math.Float32frombits(binary.LittleEndian.Uint32(b[12:16]))
}

func (v *ValueCFrameQuat) FromBytes(b []byte) error {
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

// ToCFrame converts the value to a ValueCFrame.
func (v ValueCFrameQuat) ToCFrame() ValueCFrame {
	if v.Special != 0 {
		return ValueCFrame{
			Special:  v.Special,
			Position: v.Position,
		}
	}
	return ValueCFrame{
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

type ValueToken uint32

func (ValueToken) Type() typeID {
	return typeToken
}

func (v ValueToken) BytesLen() int {
	return zToken
}

func (v ValueToken) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, uint32(v))
}

func (v *ValueToken) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = ValueToken(binary.BigEndian.Uint32(b))
	return nil
}

////////////////////////////////////////////////////////////////

const zReference = zi32

type ValueReference int32

func (ValueReference) Type() typeID {
	return typeReference
}

func (v ValueReference) BytesLen() int {
	return zReference
}

func (v ValueReference) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, encodeZigzag32(int32(v)))
}

func (v *ValueReference) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = ValueReference(decodeZigzag32(binary.BigEndian.Uint32(b)))
	return nil
}

////////////////////////////////////////////////////////////////

const zVector3int16 = zi16 * 3

type ValueVector3int16 struct {
	X, Y, Z int16
}

func (ValueVector3int16) Type() typeID {
	return typeVector3int16
}

func (v ValueVector3int16) BytesLen() int {
	return zVector3int16
}

func (v ValueVector3int16) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint16(b[0:2], uint16(v.X))
	binary.LittleEndian.PutUint16(b[2:4], uint16(v.Y))
	binary.LittleEndian.PutUint16(b[4:6], uint16(v.Z))
}

func (v *ValueVector3int16) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.X = int16(binary.LittleEndian.Uint16(b[0:2]))
	v.Y = int16(binary.LittleEndian.Uint16(b[2:4]))
	v.Z = int16(binary.LittleEndian.Uint16(b[4:6]))
	return nil
}

////////////////////////////////////////////////////////////////

const zNumberSequenceKeypoint = zf32 * 3

type ValueNumberSequenceKeypoint struct {
	Time, Value, Envelope float32
}

const zNumberSequence = zVar

type ValueNumberSequence []ValueNumberSequenceKeypoint

func (ValueNumberSequence) Type() typeID {
	return typeNumberSequence
}

func (v ValueNumberSequence) BytesLen() int {
	return zArrayLen + zNumberSequenceKeypoint*len(v)
}

func (v ValueNumberSequence) Bytes(b []byte) {
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

func (v *ValueNumberSequence) FromBytes(b []byte) error {
	b, n, err := checkvarlen(v, b)
	if err != nil {
		return err
	}
	a := make(ValueNumberSequence, n)
	for i := 0; i < n; i++ {
		bk := b[i*zNumberSequenceKeypoint:]
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

const zColorSequenceKeypoint = zf32 + zColor3 + zf32

type ValueColorSequenceKeypoint struct {
	Time     float32
	Value    ValueColor3
	Envelope float32
}

const zColorSequence = zVar

type ValueColorSequence []ValueColorSequenceKeypoint

func (ValueColorSequence) Type() typeID {
	return typeColorSequence
}

func (v ValueColorSequence) BytesLen() int {
	return zArrayLen + zColorSequenceKeypoint*len(v)
}

func (v ValueColorSequence) Bytes(b []byte) {
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

func (v *ValueColorSequence) FromBytes(b []byte) error {
	b, n, err := checkvarlen(v, b)
	if err != nil {
		return err
	}
	a := make(ValueColorSequence, n)
	for i := 0; i < n; i++ {
		bk := b[i*zColorSequenceKeypoint:]
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

const zNumberRange = zf32 * 2

type ValueNumberRange struct {
	Min, Max float32
}

func (ValueNumberRange) Type() typeID {
	return typeNumberRange
}

func (v ValueNumberRange) BytesLen() int {
	return zNumberRange
}

func (v ValueNumberRange) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(v.Min))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(v.Max))
}

func (v *ValueNumberRange) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.Min = math.Float32frombits(binary.LittleEndian.Uint32(b[0:4]))
	v.Max = math.Float32frombits(binary.LittleEndian.Uint32(b[4:8]))
	return nil
}

////////////////////////////////////////////////////////////////

const zRect = zVector2 * 2

type ValueRect struct {
	Min, Max ValueVector2
}

func (ValueRect) Type() typeID {
	return typeRect
}

func (v ValueRect) BytesLen() int {
	return zRect
}

func (v ValueRect) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	v.Min.Bytes(b[0:8])
	v.Max.Bytes(b[8:16])
}

func (v *ValueRect) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	v.Min.FromBytes(b[0:8])
	v.Max.FromBytes(b[8:16])
	return nil
}

func (ValueRect) fieldLen() []int {
	return []int{4, 4, 4, 4}
}

func (v *ValueRect) fieldSet(i int, b []byte) (err error) {
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

func (v ValueRect) fieldGet(i int, b []byte) {
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

type ValuePhysicalProperties struct {
	CustomPhysics    byte
	Density          float32
	Friction         float32
	Elasticity       float32
	FrictionWeight   float32
	ElasticityWeight float32
}

func (ValuePhysicalProperties) Type() typeID {
	return typePhysicalProperties
}

func (v ValuePhysicalProperties) BytesLen() int {
	if v.CustomPhysics == 0 {
		return zPhysicalPropertiesShort
	}
	return zPhysicalPropertiesFull
}

func (v ValuePhysicalProperties) ppBytes(b []byte) {
	binary.LittleEndian.PutUint32(b[0*zf32:0*zf32+zf32], math.Float32bits(v.Density))
	binary.LittleEndian.PutUint32(b[1*zf32:1*zf32+zf32], math.Float32bits(v.Friction))
	binary.LittleEndian.PutUint32(b[2*zf32:2*zf32+zf32], math.Float32bits(v.Elasticity))
	binary.LittleEndian.PutUint32(b[3*zf32:3*zf32+zf32], math.Float32bits(v.FrictionWeight))
	binary.LittleEndian.PutUint32(b[4*zf32:4*zf32+zf32], math.Float32bits(v.ElasticityWeight))
}

func (v ValuePhysicalProperties) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	if v.CustomPhysics == 0 {
		b[0] = 1
		return
	}
	b[0] = v.CustomPhysics
	v.ppBytes(b[zPhysicalPropertiesCP:])
}

func (v *ValuePhysicalProperties) ppFromBytes(b []byte) {
	v.Density = math.Float32frombits(binary.LittleEndian.Uint32(b[0*zf32 : 0*zf32+zf32]))
	v.Friction = math.Float32frombits(binary.LittleEndian.Uint32(b[1*zf32 : 1*zf32+zf32]))
	v.Elasticity = math.Float32frombits(binary.LittleEndian.Uint32(b[2*zf32 : 2*zf32+zf32]))
	v.FrictionWeight = math.Float32frombits(binary.LittleEndian.Uint32(b[3*zf32 : 3*zf32+zf32]))
	v.ElasticityWeight = math.Float32frombits(binary.LittleEndian.Uint32(b[4*zf32 : 4*zf32+zf32]))
}

func (v *ValuePhysicalProperties) FromBytes(b []byte) error {
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

////////////////////////////////////////////////////////////////

const zColor3uint8 = zb * 3

type ValueColor3uint8 struct {
	R, G, B byte
}

func (ValueColor3uint8) Type() typeID {
	return typeColor3uint8
}

func (v ValueColor3uint8) BytesLen() int {
	return zColor3uint8
}

func (v ValueColor3uint8) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	b[0] = v.R
	b[1] = v.G
	b[2] = v.B
}

func (v *ValueColor3uint8) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
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

func (v ValueColor3uint8) fieldGet(i int, b []byte) {
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

type ValueInt64 int64

func (ValueInt64) Type() typeID {
	return typeInt64
}

func (v ValueInt64) BytesLen() int {
	return zInt64
}

func (v ValueInt64) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint64(b, encodeZigzag64(int64(v)))
}

func (v *ValueInt64) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = ValueInt64(decodeZigzag64(binary.BigEndian.Uint64(b)))
	return nil
}

////////////////////////////////////////////////////////////////

const zSharedString = zu32

type ValueSharedString uint32

func (ValueSharedString) Type() typeID {
	return typeSharedString
}

func (v ValueSharedString) BytesLen() int {
	return zSharedString
}

func (v ValueSharedString) Bytes(b []byte) {
	_ = b[v.BytesLen()-1]
	binary.BigEndian.PutUint32(b, uint32(v))
}

func (v *ValueSharedString) FromBytes(b []byte) error {
	if err := checklen(v, b); err != nil {
		return err
	}
	*v = ValueSharedString(binary.BigEndian.Uint32(b))
	return nil
}

////////////////////////////////////////////////////////////////
