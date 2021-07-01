package rbxl

import (
	"encoding/binary"
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

// Encodes and decodes a Value based on its fields
type fielder interface {
	// Value.Type
	Type() typeID
	// Length of each field
	fieldLen() []int
	// Set bytes of nth field
	fieldSet(int, []byte) error
	// Get bytes of nth field
	fieldGet(int, []byte)
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

// Encodes Values that implement the fielder interface.
func interleaveFields(id typeID, a []value) (b []byte, err error) {
	if len(a) == 0 {
		return b, nil
	}

	af := make([]fielder, len(a))
	for i, v := range a {
		af[i] = v.(fielder)
		if af[i].Type() != id {
			return nil, errElementType{Index: i, Got: af[i].Type().String(), Want: id.String()}
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
	ofields := make([]int, maxFieldLen+1)[:len(nbytes)+1]
	for i, n := range nbytes {
		tbytes += n
		ofields[i+1] = ofields[i] + n*nvalues
	}

	b = make([]byte, tbytes*nvalues)

	// List of each field slice
	fields := make([][]byte, maxFieldLen)[:nfields]
	for i := range fields {
		// Each field slice affects the final array
		fields[i] = b[ofields[i]:ofields[i+1]]
	}

	for i, v := range af {
		for f, field := range fields {
			v.fieldGet(f, field[i*nbytes[f]:])
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

// Appends the bytes of a list of Values into a byte array.
func appendValueBytes(id typeID, a []value) []byte {
	n := 0
	for _, v := range a {
		n += v.BytesLen()
	}
	b := make([]byte, n)
	c := b[:]
	for _, v := range a {
		n := v.BytesLen()
		v.Bytes(c[:n])
		c = c[n:]
	}
	return b
}

// Append each value as bytes, then interleave to improve compression.
func interleaveAppend(t typeID, a []value) (b []byte, err error) {
	b = appendValueBytes(t, a)
	if err = interleave(b, t.Size()); err != nil {
		return nil, err
	}
	return b, nil
}

// valuesToBytes encodes a slice of values into binary form, according to t.
// Returns an error if a value cannot be encoded as t.
func valuesToBytes(t typeID, a []value) (b []byte, err error) {
	if !t.Valid() {
		return nil, errInvalidType(t)
	}
	for i, v := range a {
		if v.Type() != t {
			return nil, errElementType{Index: i, Got: v.Type().String(), Want: t.String()}
		}
	}

	switch t {
	case typeString:
		return appendValueBytes(t, a), nil

	case typeBool:
		return appendValueBytes(t, a), nil

	case typeInt:
		return interleaveAppend(t, a)

	case typeFloat:
		return interleaveAppend(t, a)

	case typeDouble:
		return appendValueBytes(t, a), nil

	case typeUDim:
		return interleaveFields(t, a)

	case typeUDim2:
		return interleaveFields(t, a)

	case typeRay:
		return appendValueBytes(t, a), nil

	case typeFaces:
		return appendValueBytes(t, a), nil

	case typeAxes:
		return appendValueBytes(t, a), nil

	case typeBrickColor:
		return interleaveAppend(t, a)

	case typeColor3:
		return interleaveFields(t, a)

	case typeVector2:
		return interleaveFields(t, a)

	case typeVector3:
		return interleaveFields(t, a)

	case typeVector2int16:
		return nil, errors.New("not implemented")

	case typeCFrame:
		// The bytes of each value can vary in length.
		p := make([]value, len(a))
		for i, cf := range a {
			cf := cf.(*valueCFrame)
			// Build matrix part.
			b = append(b, cf.Special)
			if cf.Special == 0 {
				// Write all components.
				r := make([]byte, len(cf.Rotation)*zf32)
				for i, f := range cf.Rotation {
					binary.LittleEndian.PutUint32(r[i*zf32:i*zf32+zf32], math.Float32bits(f))
				}
				b = append(b, r...)
			}
			// Prepare position part.
			p[i] = &cf.Position
		}
		// Build position part.
		pb, _ := interleaveFields(typeVector3, p)
		b = append(b, pb...)
		return b, nil

	case typeCFrameQuat:
		p := make([]value, len(a))
		for i, cf := range a {
			cf := cf.(*valueCFrameQuat)
			b = append(b, cf.Special)
			if cf.Special == 0 {
				r := make([]byte, zCFrameQuatQ)
				cf.quatBytes(r)
				b = append(b, r...)
			}
			p[i] = &cf.Position
		}
		pb, _ := interleaveFields(typeVector3, p)
		b = append(b, pb...)
		return b, nil

	case typeToken:
		return interleaveAppend(t, a)

	case typeReference:
		// Because values are generated in sequence, they are likely to be
		// relatively close to each other. Subtracting each value from the
		// previous will likely produce small values that compress well.
		if len(a) == 0 {
			return b, nil
		}
		b = make([]byte, len(a)*zReference)
		var prev valueReference
		for i, ref := range a {
			ref := ref.(*valueReference)
			if i == 0 {
				ref.Bytes(b[i*zReference : i*zReference+zReference])
			} else {
				// Convert absolute ref to relative ref.
				(*ref - prev).Bytes(b[i*zReference : i*zReference+zReference])
			}
			prev = *ref
		}
		if err = interleave(b, zReference); err != nil {
			return nil, err
		}
		return b, nil

	case typeVector3int16:
		return appendValueBytes(t, a), nil

	case typeNumberSequence:
		return appendValueBytes(t, a), nil

	case typeColorSequence:
		return appendValueBytes(t, a), nil

	case typeNumberRange:
		return appendValueBytes(t, a), nil

	case typeRect:
		return interleaveFields(t, a)

	case typePhysicalProperties:
		// The bytes of each value can vary in length.
		q := make([]byte, zPhysicalPropertiesFields)
		for _, pp := range a {
			pp := pp.(*valuePhysicalProperties)
			b = append(b, pp.CustomPhysics)
			if pp.CustomPhysics != 0 {
				// Write all fields.
				pp.ppBytes(q)
				b = append(b, q...)
			}
		}
		return b, nil

	case typeColor3uint8:
		return interleaveFields(t, a)

	case typeInt64:
		return interleaveAppend(t, a)

	case typeSharedString:
		return interleaveAppend(t, a)

	default:
		return b, nil
	}
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

// Decodes Values that implement the fielder interface.
func deinterleaveFields(id typeID, b []byte) (a []value, err error) {
	if len(b) == 0 {
		return a, nil
	}

	if !id.Valid() {
		return nil, fmt.Errorf("type identifier 0x%X is not a valid Type.", id)
	}

	// Number of bytes per field
	nbytes := newValue(id).(fielder).fieldLen()
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
	// Offset of each field slice.
	ofields := make([]int, maxFieldLen+1)[:len(nbytes)+1]
	for i, n := range nbytes {
		ofields[i+1] = ofields[i] + n*nvalues
	}

	a = make([]value, nvalues)

	// List of each field slice
	fields := make([][]byte, maxFieldLen)[:nfields]
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
		v := newValue(id)
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

// appendByteValues reads a byte array as an array of Values of a certain type.
// If id.Size() is less than 0, then values are assumed to be of variable
// length. The first 4 bytes of a value is read as length N of the value.
// id.FieldSize() indicates the size of each field in the value, so the next
// N*field bytes are read as the full value.
func appendByteValues(id typeID, b []byte) (a []value, err error) {
	if size := id.Size(); size >= 0 {
		for i := 0; i+size <= len(b); i += size {
			v := newValue(id)
			if err := v.FromBytes(b[i : i+size]); err != nil {
				return nil, err
			}
			a = append(a, v)
		}
		return a, nil
	}
	// Variable length; get size from first 4 bytes.
	field := id.FieldSize()
	ba := b
	for len(ba) > 0 {
		if len(ba) < zArrayLen {
			return nil, errExpectedMoreBytes(4)
		}
		size := int(binary.LittleEndian.Uint32(ba))
		if len(ba[zArrayLen:]) < size*field {
			return nil, errExpectedMoreBytes(size * field)
		}

		v := newValue(id)
		if err := v.FromBytes(ba[:zArrayLen+size*field]); err != nil {
			return nil, err
		}
		a = append(a, v)

		ba = ba[zArrayLen+size*field:]
	}
	return a, nil
}

// Deinterleave, then append from given size.
func deinterleaveAppend(t typeID, b []byte) (a []value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	size := t.Size()
	if err = deinterleave(bc, size); err != nil {
		return nil, err
	}
	return appendByteValues(t, bc)
}

// valuesFromBytes decodes b according to t, into a slice of values, the type of
// each corresponding to t.
func valuesFromBytes(t typeID, b []byte) (a []value, err error) {
	if !t.Valid() {
		return nil, errInvalidType(t)
	}

	switch t {
	case typeString:
		return appendByteValues(t, b)

	case typeBool:
		return appendByteValues(t, b)

	case typeInt:
		return deinterleaveAppend(t, b)

	case typeFloat:
		return deinterleaveAppend(t, b)

	case typeDouble:
		return appendByteValues(t, b)

	case typeUDim:
		return deinterleaveFields(t, b)

	case typeUDim2:
		return deinterleaveFields(t, b)

	case typeRay:
		return appendByteValues(t, b)

	case typeFaces:
		return appendByteValues(t, b)

	case typeAxes:
		return appendByteValues(t, b)

	case typeBrickColor:
		return deinterleaveAppend(t, b)

	case typeColor3:
		return deinterleaveFields(t, b)

	case typeVector2:
		return deinterleaveFields(t, b)

	case typeVector3:
		return deinterleaveFields(t, b)

	case typeVector2int16:
		return nil, errors.New("not implemented")

	case typeCFrame:
		cfs := make([]*valueCFrame, 0)
		// This loop reads the matrix data. i is the current position in the
		// byte array. n is the expected size of the position data, which
		// increases every time another CFrame is read. As long as the number of
		// remaining bytes is greater than n, then the next byte can be assumed
		// to be matrix data.
		i := 0
		n := 0
		for ; len(b)-i > n; n += zVector3 {
			cf := new(valueCFrame)
			cf.Special = b[i]
			i++
			if cf.Special == 0 {
				q := len(cf.Rotation) * zf32
				r := b[i:]
				if len(r) < q {
					return nil, errExpectedMoreBytes(q)
				}
				for i := range cf.Rotation {
					cf.Rotation[i] = math.Float32frombits(binary.LittleEndian.Uint32(r[i*zf32 : i*zf32+zf32]))
				}
				i += q
			}
			cfs = append(cfs, cf)
		}
		// Read remaining position data using the Position field, which is a
		// valueVector3.
		if a, err = deinterleaveFields(typeVector3, b[i:i+n]); err != nil {
			return nil, err
		}
		if len(a) != len(cfs) {
			return nil, errors.New("number of positions does not match number of matrices")
		}
		// Hack: use 'a' variable to receive Vector3 values, then replace them
		// with CFrames. This lets us avoid needing to copy 'cfs' to 'a', and
		// needing to create a second array.
		for i, p := range a {
			cfs[i].Position = *p.(*valueVector3)
			a[i] = cfs[i]
		}
		return a, nil

	case typeCFrameQuat:
		cfs := make([]*valueCFrame, 0)
		i := 0
		n := 0
		for ; len(b)-i > n; n += zVector3 {
			cf := new(valueCFrameQuat)
			cf.Special = b[i]
			i++
			if cf.Special == 0 {
				r := b[i:]
				if len(r) < zCFrameQuatQ {
					return nil, errExpectedMoreBytes(zCFrameQuatQ)
				}
				cf.quatFromBytes(r)
				i += zCFrameQuatQ
			}
			c := cf.ToCFrame()
			cfs = append(cfs, &c)
		}
		if a, err = deinterleaveFields(typeVector3, b[i:i+n]); err != nil {
			return nil, err
		}
		if len(a) != len(cfs) {
			return nil, errors.New("number of positions does not match number of matrices")
		}
		for i, p := range a {
			cfs[i].Position = *p.(*valueVector3)
			a[i] = cfs[i]
		}
		return a, nil

	case typeToken:
		return deinterleaveAppend(t, b)

	case typeReference:
		if len(b) == 0 {
			return a, nil
		}
		if len(b)%zReference != 0 {
			return nil, fmt.Errorf("array must be divisible by %d", zReference)
		}
		bc := make([]byte, len(b))
		copy(bc, b)
		if err = deinterleave(bc, zReference); err != nil {
			return nil, err
		}
		a = make([]value, len(bc)/zReference)
		for i := 0; i < len(bc)/zReference; i++ {
			ref := new(valueReference)
			ref.FromBytes(bc[i*zReference : i*zReference+zReference])
			if i > 0 {
				// Convert relative ref to absolute ref.
				r := *a[i-1].(*valueReference)
				*ref = r + *ref
			}
			a[i] = ref
		}
		return a, nil

	case typeVector3int16:
		return appendByteValues(t, b)

	case typeNumberSequence:
		return appendByteValues(t, b)

	case typeColorSequence:
		return appendByteValues(t, b)

	case typeNumberRange:
		return appendByteValues(t, b)

	case typeRect:
		return deinterleaveFields(t, b)

	case typePhysicalProperties:
		for i := 0; i < len(b); {
			pp := new(valuePhysicalProperties)
			pp.CustomPhysics = b[i]
			i++
			if pp.CustomPhysics != 0 {
				if len(b[i:]) < zPhysicalPropertiesFields {
					return nil, errExpectedMoreBytes(zPhysicalPropertiesFields)
				}
				pp.ppFromBytes(b[i:])
				i += zPhysicalPropertiesFields
			}
			a = append(a, pp)
		}
		return a, nil

	case typeColor3uint8:
		return deinterleaveFields(t, b)

	case typeInt64:
		return deinterleaveAppend(t, b)

	case typeSharedString:
		return deinterleaveAppend(t, b)

	default:
		return a, nil
	}
}
