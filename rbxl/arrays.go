package rbxl

import (
	"errors"
	"fmt"
	"math"
)

type errExpectedMoreBytes int

func (err errExpectedMoreBytes) Error() string {
	return fmt.Sprintf("expected %d more bytes in array", int(err))
}

type errElementType struct {
	Index int
	Got   string
	Want  string
}

func (err errElementType) Error() string {
	return fmt.Sprintf("element %d is of type %s where %s is expected", err.Index, err.Got, err.Want)
}

type errInvalidType byte

func (err errInvalidType) Error() string {
	return fmt.Sprintf("invalid type (%02X)", byte(err))
}

type indexError struct {
	Index int
	Cause error
}

func (err indexError) Error() string {
	return fmt.Sprintf("#%d: %s", err.Index, err.Cause)
}

func (err indexError) Unwrap() error {
	return err.Cause
}

// Interleave transforms an array of bytes by interleaving them based on a
// given size. The size must be a divisor of the array length.
//
// The array is divided into groups, each `length` in size. The nth elements
// of each group are then moved so that they are group together. For example:
//
//     Original:    ABCDabcd
//     Interleaved: AaBbCcDd
func interleave(bytes []byte, length int) error {
	if length <= 0 {
		return errors.New("length must be greater than 0")
	}
	if len(bytes)%length != 0 {
		return fmt.Errorf("length (%d) must be a divisor of array length (%d)", length, len(bytes))
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
	loop:
		for start := range bytes {
			next := (start%rows)*cols + start/rows
			if next <= start {
				continue loop
			}
			for {
				if next = (next%rows)*cols + next/rows; next < start {
					continue loop
				} else if next == start {
					break
				}
			}
			for next, tmp := start, bytes[start]; ; {
				i := (next%rows)*cols + next/rows
				if i == start {
					bytes[next] = tmp
				} else {
					bytes[next] = bytes[i]
				}
				if next = i; next <= start {
					break
				}
			}
		}
	}
	return nil
}

func deinterleave(bytes []byte, size int) error {
	if size <= 0 {
		return errors.New("size must be greater than 0")
	}
	if len(bytes)%size != 0 {
		return fmt.Errorf("size (%d) must be a divisor of array length (%d)", size, len(bytes))
	}

	return interleave(bytes, len(bytes)/size)
}

func arrayToBytes(b []byte, a array) (r []byte, err error) {
	r = a.Bytes(b)

	if _, ok := a.(interleaver); ok {
		size := a.Type().Size()
		if size <= 0 {
			panic("interleaving non-constant type size")
		}
		// Only interleave the extended part.
		if err := interleave(r[len(b):], size); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func typeArrayToBytes(a array) (b []byte, err error) {
	b = make([]byte, 0, zb+a.BytesLen())
	b = append(b, byte(a.Type()))
	return arrayToBytes(b, a)
}

func refArrayToBytes(refs []int32) (b []byte) {
	a := make(arrayReference, len(refs))
	for i, r := range refs {
		a[i] = valueReference(r)
	}
	b = make([]byte, 0, a.BytesLen())
	b, _ = arrayToBytes(b, a)
	return b
}

// arrayFromBytes decodes an array of length elements from b into a. Returns an
// error if the array could not be decoded. n is the number of bytes
// successfully read from b.
func arrayFromBytes(b []byte, a array) (n int, err error) {
	if _, ok := a.(interleaver); ok {
		size := a.Type().Size()
		if size <= 0 {
			panic("deinterleaving non-constant type size")
		}
		if err := deinterleave(b, size); err != nil {
			return 0, err
		}
	}

	if a, ok := a.(fromByter); ok {
		if n, err = a.FromBytes(b); err != nil {
			return n, err
		}
		return n, nil
	}

	for i := 0; i < a.Len(); i++ {
		v := newValue(a.Type())
		nn, err := v.FromBytes(b)
		if err != nil {
			return n, indexError{Index: i, Cause: err}
		}
		n += nn
		b = b[nn:]
		a.Set(i, v)
	}
	return n, nil
}

func typeArrayFromBytes(b []byte, length int) (a array, n int, err error) {
	if len(b) < zb {
		return nil, 0, buflenError{exp: zb, got: len(b)}
	}
	t := typeID(b[0])
	n += zb
	if !t.Valid() {
		return a, n, errUnknownType(t)
	}
	b = b[n:]

	a = newArray(t, length)

	nn, err := arrayFromBytes(b, a)
	if err != nil {
		return a, n, err
	}
	n += nn
	return a, n, nil
}

func refArrayFromBytes(b []byte, length int) (a arrayReference, err error) {
	a = make(arrayReference, length)
	_, err = arrayFromBytes(b, a)
	return a, err
}

////////////////////////////////////////////////////////////////////////////////

type array interface {
	// Type returns an identifier indicating the type of each value in the
	// array.
	Type() typeID

	// Len returns the length of the array.
	Len() int

	// Get gets index i to v. Panics if i is out of bounds.
	Get(i int) value

	// Set sets index i to v. Panics if i is out of bounds, or v is not of the
	// array's type.
	Set(i int, v value)

	// BytesLen returns the number of bytes required to encode the array.
	BytesLen() int

	// Bytes encodes the array, appending to b. Returns the extended buffer.
	Bytes(b []byte) []byte
}

// interleaver indicates that the encoded bytes of the array are interleaved.
// The type of an interleaver must have a constant size.
type interleaver interface {
	array
	Interleaved()
}

// fromByter is an array with a custom decoding implementation.
type fromByter interface {
	array
	FromBytes(b []byte) (n int, err error)
}

func newArray(t typeID, n int) array {
	switch t {
	case typeString:
		return make(arrayString, n)
	case typeBool:
		return make(arrayBool, n)
	case typeInt:
		return make(arrayInt, n)
	case typeFloat:
		return make(arrayFloat, n)
	case typeDouble:
		return make(arrayDouble, n)
	case typeUDim:
		return make(arrayUDim, n)
	case typeUDim2:
		return make(arrayUDim2, n)
	case typeRay:
		return make(arrayRay, n)
	case typeFaces:
		return make(arrayFaces, n)
	case typeAxes:
		return make(arrayAxes, n)
	case typeBrickColor:
		return make(arrayBrickColor, n)
	case typeColor3:
		return make(arrayColor3, n)
	case typeVector2:
		return make(arrayVector2, n)
	case typeVector3:
		return make(arrayVector3, n)
	case typeVector2int16:
		return make(arrayVector2int16, n)
	case typeCFrame:
		return make(arrayCFrame, n)
	case typeCFrameQuat:
		return make(arrayCFrameQuat, n)
	case typeToken:
		return make(arrayToken, n)
	case typeReference:
		return make(arrayReference, n)
	case typeVector3int16:
		return make(arrayVector3int16, n)
	case typeNumberSequence:
		return make(arrayNumberSequence, n)
	case typeColorSequence:
		return make(arrayColorSequence, n)
	case typeNumberRange:
		return make(arrayNumberRange, n)
	case typeRect:
		return make(arrayRect, n)
	case typePhysicalProperties:
		return make(arrayPhysicalProperties, n)
	case typeColor3uint8:
		return make(arrayColor3uint8, n)
	case typeInt64:
		return make(arrayInt64, n)
	case typeSharedString:
		return make(arraySharedString, n)
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type arrayString []valueString

func (arrayString) Type() typeID {
	return typeString
}

func (a arrayString) Len() int {
	return len(a)
}

func (a arrayString) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayString) Set(i int, v value) {
	a[i] = *v.(*valueString)
}

func (a arrayString) BytesLen() int {
	var n int
	for _, v := range a {
		n += v.BytesLen()
	}
	return n
}

func (a arrayString) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayBool []valueBool

func (arrayBool) Type() typeID {
	return typeBool
}

func (a arrayBool) Len() int {
	return len(a)
}

func (a arrayBool) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayBool) Set(i int, v value) {
	a[i] = *v.(*valueBool)
}

func (a arrayBool) BytesLen() int {
	return len(a) * zBool
}

func (a arrayBool) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayInt []valueInt

func (arrayInt) Type() typeID {
	return typeInt
}

func (a arrayInt) Len() int {
	return len(a)
}

func (a arrayInt) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayInt) Set(i int, v value) {
	a[i] = *v.(*valueInt)
}

func (a arrayInt) BytesLen() int {
	return len(a) * zInt
}

func (a arrayInt) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayInt) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayFloat []valueFloat

func (arrayFloat) Type() typeID {
	return typeFloat
}

func (a arrayFloat) Len() int {
	return len(a)
}

func (a arrayFloat) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayFloat) Set(i int, v value) {
	a[i] = *v.(*valueFloat)
}

func (a arrayFloat) BytesLen() int {
	return len(a) * zFloat
}

func (a arrayFloat) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayFloat) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayDouble []valueDouble

func (arrayDouble) Type() typeID {
	return typeDouble
}

func (a arrayDouble) Len() int {
	return len(a)
}

func (a arrayDouble) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayDouble) Set(i int, v value) {
	a[i] = *v.(*valueDouble)
}

func (a arrayDouble) BytesLen() int {
	return len(a) * zDouble
}

func (a arrayDouble) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayUDim []valueUDim

func (arrayUDim) Type() typeID {
	return typeUDim
}

func (a arrayUDim) Len() int {
	return len(a)
}

func (a arrayUDim) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayUDim) Set(i int, v value) {
	a[i] = *v.(*valueUDim)
}

func (a arrayUDim) BytesLen() int {
	return len(a) * zUDim
}

func (a arrayUDim) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayUDim) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayUDim2 []valueUDim2

func (arrayUDim2) Type() typeID {
	return typeUDim2
}

func (a arrayUDim2) Len() int {
	return len(a)
}

func (a arrayUDim2) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayUDim2) Set(i int, v value) {
	a[i] = *v.(*valueUDim2)
}

func (a arrayUDim2) BytesLen() int {
	return len(a) * zUDim2
}

func (a arrayUDim2) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayUDim2) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayRay []valueRay

func (arrayRay) Type() typeID {
	return typeRay
}

func (a arrayRay) Len() int {
	return len(a)
}

func (a arrayRay) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayRay) Set(i int, v value) {
	a[i] = *v.(*valueRay)
}

func (a arrayRay) BytesLen() int {
	return len(a) * zRay
}

func (a arrayRay) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayFaces []valueFaces

func (arrayFaces) Type() typeID {
	return typeFaces
}

func (a arrayFaces) Len() int {
	return len(a)
}

func (a arrayFaces) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayFaces) Set(i int, v value) {
	a[i] = *v.(*valueFaces)
}

func (a arrayFaces) BytesLen() int {
	return len(a) * zFaces
}

func (a arrayFaces) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayAxes []valueAxes

func (arrayAxes) Type() typeID {
	return typeAxes
}

func (a arrayAxes) Len() int {
	return len(a)
}

func (a arrayAxes) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayAxes) Set(i int, v value) {
	a[i] = *v.(*valueAxes)
}

func (a arrayAxes) BytesLen() int {
	return len(a) * zAxes
}

func (a arrayAxes) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayBrickColor []valueBrickColor

func (arrayBrickColor) Type() typeID {
	return typeBrickColor
}

func (a arrayBrickColor) Len() int {
	return len(a)
}

func (a arrayBrickColor) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayBrickColor) Set(i int, v value) {
	a[i] = *v.(*valueBrickColor)
}

func (a arrayBrickColor) BytesLen() int {
	return len(a) * zBrickColor
}

func (a arrayBrickColor) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayBrickColor) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayColor3 []valueColor3

func (arrayColor3) Type() typeID {
	return typeColor3
}

func (a arrayColor3) Len() int {
	return len(a)
}

func (a arrayColor3) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayColor3) Set(i int, v value) {
	a[i] = *v.(*valueColor3)
}

func (a arrayColor3) BytesLen() int {
	return len(a) * zColor3
}

func (a arrayColor3) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayColor3) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayVector2 []valueVector2

func (arrayVector2) Type() typeID {
	return typeVector2
}

func (a arrayVector2) Len() int {
	return len(a)
}

func (a arrayVector2) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayVector2) Set(i int, v value) {
	a[i] = *v.(*valueVector2)
}

func (a arrayVector2) BytesLen() int {
	return len(a) * zVector2
}

func (a arrayVector2) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayVector2) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayVector3 []valueVector3

func (arrayVector3) Type() typeID {
	return typeVector3
}

func (a arrayVector3) Len() int {
	return len(a)
}

func (a arrayVector3) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayVector3) Set(i int, v value) {
	a[i] = *v.(*valueVector3)
}

func (a arrayVector3) BytesLen() int {
	return len(a) * zVector3
}

func (a arrayVector3) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayVector3) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayVector2int16 []valueVector2int16

func (arrayVector2int16) Type() typeID {
	return typeVector2int16
}

func (a arrayVector2int16) Len() int {
	return len(a)
}

func (a arrayVector2int16) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayVector2int16) Set(i int, v value) {
	a[i] = *v.(*valueVector2int16)
}

func (a arrayVector2int16) BytesLen() int {
	return len(a) * zVector2int16
}

func (a arrayVector2int16) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayCFrame []valueCFrame

func (arrayCFrame) Type() typeID {
	return typeCFrame
}

func (a arrayCFrame) Len() int {
	return len(a)
}

func (a arrayCFrame) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayCFrame) Set(i int, v value) {
	a[i] = *v.(*valueCFrame)
}

func (a arrayCFrame) BytesLen() int {
	var n int
	for _, v := range a {
		n += v.BytesLen()
	}
	return n
}

func (a arrayCFrame) Bytes(b []byte) []byte {
	p := make([]byte, 0, len(a)*zVector3)
	for _, v := range a {
		b = append(b, v.Special)
		if v.Special == 0 {
			for _, f := range v.Rotation {
				b = appendUint32(b, le, math.Float32bits(f))
			}
		}
		p = v.Position.Bytes(p)
	}
	interleave(p, zVector3)
	b = append(b, p...)
	return b
}

func (a arrayCFrame) FromBytes(b []byte) (n int, err error) {
	for i := range a {
		var cond byte
		var nn int
		cond, b, nn, err = checkLengthCond(&a[i], b)
		if err != nil {
			return n, indexError{Index: i, Cause: err}
		}
		n += nn
		a[i].Special = cond
		if cond == 0 {
			n += zCFrameRo
			for j := range a[i].Rotation {
				a[i].Rotation[j] = math.Float32frombits(readUint32(&b, le))
			}
		} else {
			for j := range a[i].Rotation {
				a[i].Rotation[j] = 0
			}
		}
	}
	if err := deinterleave(b, zVector3); err != nil {
		return n, err
	}
	for i := range a {
		var v valueVector3
		nn, err := v.FromBytes(b)
		if err != nil {
			return n, indexError{Index: i, Cause: err}
		}
		n += nn
		b = b[nn:]
		a[i].Position = v
	}
	return n, nil
}

////////////////////////////////////////////////////////////////////////////////

type arrayCFrameQuat []valueCFrameQuat

func (arrayCFrameQuat) Type() typeID {
	return typeCFrameQuat
}

func (a arrayCFrameQuat) Len() int {
	return len(a)
}

func (a arrayCFrameQuat) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayCFrameQuat) Set(i int, v value) {
	a[i] = *v.(*valueCFrameQuat)
}

func (a arrayCFrameQuat) BytesLen() int {
	var n int
	for _, v := range a {
		n += v.BytesLen()
	}
	return n
}

func (a arrayCFrameQuat) Bytes(b []byte) []byte {
	p := make([]byte, 0, len(a)*zVector3)
	for _, v := range a {
		b = append(b, v.Special)
		if v.Special == 0 {
			b = v.quatBytes(b)
		}
		p = v.Position.Bytes(p)
	}
	interleave(p, zVector3)
	b = append(b, p...)
	return b
}

func (a arrayCFrameQuat) FromBytes(b []byte) (n int, err error) {
	for i := range a {
		var cond byte
		var nn int
		cond, b, nn, err = checkLengthCond(&a[i], b)
		if err != nil {
			return n, indexError{Index: i, Cause: err}
		}
		n += nn
		a[i].Special = cond
		if cond == 0 {
			n += zCFrameQuatQ
			b = a[i].quatFromBytes(b)
		} else {
			a[i].QX = 0
			a[i].QY = 0
			a[i].QZ = 0
			a[i].QW = 0
		}
	}
	if err := deinterleave(b, zVector3); err != nil {
		return n, err
	}
	for i := range a {
		var v valueVector3
		nn, err := v.FromBytes(b)
		if err != nil {
			return n, indexError{Index: i, Cause: err}
		}
		n += nn
		b = b[nn:]
		a[i].Position = v
	}
	return n, nil
}

////////////////////////////////////////////////////////////////////////////////

type arrayToken []valueToken

func (arrayToken) Type() typeID {
	return typeToken
}

func (a arrayToken) Len() int {
	return len(a)
}

func (a arrayToken) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayToken) Set(i int, v value) {
	a[i] = *v.(*valueToken)
}

func (a arrayToken) BytesLen() int {
	return len(a) * zToken
}

func (a arrayToken) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayToken) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayReference []valueReference

func (arrayReference) Type() typeID {
	return typeReference
}

func (a arrayReference) Len() int {
	return len(a)
}

func (a arrayReference) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayReference) Set(i int, v value) {
	a[i] = *v.(*valueReference)
}

func (a arrayReference) BytesLen() int {
	return len(a) * zReference
}

func (a arrayReference) Bytes(b []byte) []byte {
	// Because values are generated in sequence, they are likely to be
	// relatively close to each other. Subtracting each value from the previous
	// will likely produce small values that compress well.
	if len(a) == 0 {
		return b
	}
	prev := a[0]
	b = prev.Bytes(b)
	for i := 1; i < len(a); i++ {
		v := a[i]
		b = (v - prev).Bytes(b)
		prev = v
	}
	return b
}

func (a arrayReference) FromBytes(b []byte) (n int, err error) {
	for i := 0; i < len(a); i++ {
		var v valueReference
		nn, err := v.FromBytes(b)
		if err != nil {
			return n, indexError{Index: i, Cause: err}
		}
		n += nn
		b = b[nn:]
		if i > 0 {
			v += a[i-1]
		}
		a[i] = v
	}
	return n, nil
}

func (a arrayReference) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayVector3int16 []valueVector3int16

func (arrayVector3int16) Type() typeID {
	return typeVector3int16
}

func (a arrayVector3int16) Len() int {
	return len(a)
}

func (a arrayVector3int16) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayVector3int16) Set(i int, v value) {
	a[i] = *v.(*valueVector3int16)
}

func (a arrayVector3int16) BytesLen() int {
	return len(a) * zVector3int16
}

func (a arrayVector3int16) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayNumberSequence []valueNumberSequence

func (arrayNumberSequence) Type() typeID {
	return typeNumberSequence
}

func (a arrayNumberSequence) Len() int {
	return len(a)
}

func (a arrayNumberSequence) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayNumberSequence) Set(i int, v value) {
	a[i] = *v.(*valueNumberSequence)
}

func (a arrayNumberSequence) BytesLen() int {
	var n int
	for _, v := range a {
		n += v.BytesLen()
	}
	return n
}

func (a arrayNumberSequence) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayColorSequence []valueColorSequence

func (arrayColorSequence) Type() typeID {
	return typeColorSequence
}

func (a arrayColorSequence) Len() int {
	return len(a)
}

func (a arrayColorSequence) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayColorSequence) Set(i int, v value) {
	a[i] = *v.(*valueColorSequence)
}

func (a arrayColorSequence) BytesLen() int {
	var n int
	for _, v := range a {
		n += v.BytesLen()
	}
	return n
}

func (a arrayColorSequence) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayNumberRange []valueNumberRange

func (arrayNumberRange) Type() typeID {
	return typeNumberRange
}

func (a arrayNumberRange) Len() int {
	return len(a)
}

func (a arrayNumberRange) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayNumberRange) Set(i int, v value) {
	a[i] = *v.(*valueNumberRange)
}

func (a arrayNumberRange) BytesLen() int {
	return len(a) * zNumberRange
}

func (a arrayNumberRange) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayRect []valueRect

func (arrayRect) Type() typeID {
	return typeRect
}

func (a arrayRect) Len() int {
	return len(a)
}

func (a arrayRect) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayRect) Set(i int, v value) {
	a[i] = *v.(*valueRect)
}

func (a arrayRect) BytesLen() int {
	return len(a) * zRect
}

func (a arrayRect) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayRect) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayPhysicalProperties []valuePhysicalProperties

func (arrayPhysicalProperties) Type() typeID {
	return typePhysicalProperties
}

func (a arrayPhysicalProperties) Len() int {
	return len(a)
}

func (a arrayPhysicalProperties) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayPhysicalProperties) Set(i int, v value) {
	a[i] = *v.(*valuePhysicalProperties)
}

func (a arrayPhysicalProperties) BytesLen() int {
	var n int
	for _, v := range a {
		n += v.BytesLen()
	}
	return n
}

func (a arrayPhysicalProperties) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

////////////////////////////////////////////////////////////////////////////////

type arrayColor3uint8 []valueColor3uint8

func (arrayColor3uint8) Type() typeID {
	return typeColor3uint8
}

func (a arrayColor3uint8) Len() int {
	return len(a)
}

func (a arrayColor3uint8) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayColor3uint8) Set(i int, v value) {
	a[i] = *v.(*valueColor3uint8)
}

func (a arrayColor3uint8) BytesLen() int {
	return len(a) * zColor3uint8
}

func (a arrayColor3uint8) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayColor3uint8) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arrayInt64 []valueInt64

func (arrayInt64) Type() typeID {
	return typeInt64
}

func (a arrayInt64) Len() int {
	return len(a)
}

func (a arrayInt64) Get(i int) value {
	v := a[i]
	return &v
}

func (a arrayInt64) Set(i int, v value) {
	a[i] = *v.(*valueInt64)
}

func (a arrayInt64) BytesLen() int {
	return len(a) * zInt64
}

func (a arrayInt64) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arrayInt64) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////

type arraySharedString []valueSharedString

func (arraySharedString) Type() typeID {
	return typeSharedString
}

func (a arraySharedString) Len() int {
	return len(a)
}

func (a arraySharedString) Get(i int) value {
	v := a[i]
	return &v
}

func (a arraySharedString) Set(i int, v value) {
	a[i] = *v.(*valueSharedString)
}

func (a arraySharedString) BytesLen() int {
	return len(a) * zSharedString
}

func (a arraySharedString) Bytes(b []byte) []byte {
	for _, v := range a {
		b = v.Bytes(b)
	}
	return b
}

func (a arraySharedString) Interleaved() {}

////////////////////////////////////////////////////////////////////////////////
