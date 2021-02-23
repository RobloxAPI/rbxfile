package rbxl

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Encodes and decodes a Value based on its fields
type fielder interface {
	// Value.Type
	Type() Type
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
	ofields := make([]int, maxFieldLen)[:len(nbytes)+1]
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
func appendValueBytes(id Type, a []Value) []byte {
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
func interleaveAppend(t Type, a []Value) (b []byte, err error) {
	b = appendValueBytes(t, a)
	if err = interleave(b, t.Size()); err != nil {
		return nil, err
	}
	return b, nil
}

// ValuesToBytes encodes a slice of values into binary form, according to t.
// Returns an error if a value cannot be encoded as t.
func ValuesToBytes(t Type, a []Value) (b []byte, err error) {
	if !t.Valid() {
		return nil, fmt.Errorf("invalid type (%02X)", t)
	}
	for i, v := range a {
		if v.Type() != t {
			return nil, fmt.Errorf("element %d is of type `%s` where `%s` is expected", i, v.Type().String(), t.String())
		}
	}

	switch t {
	case TypeString:
		return appendValueBytes(t, a), nil

	case TypeBool:
		return appendValueBytes(t, a), nil

	case TypeInt:
		return interleaveAppend(t, a)

	case TypeFloat:
		return interleaveAppend(t, a)

	case TypeDouble:
		return appendValueBytes(t, a), nil

	case TypeUDim:
		return interleaveFields(t, a)

	case TypeUDim2:
		return interleaveFields(t, a)

	case TypeRay:
		return appendValueBytes(t, a), nil

	case TypeFaces:
		return appendValueBytes(t, a), nil

	case TypeAxes:
		return appendValueBytes(t, a), nil

	case TypeBrickColor:
		return interleaveAppend(t, a)

	case TypeColor3:
		return interleaveFields(t, a)

	case TypeVector2:
		return interleaveFields(t, a)

	case TypeVector3:
		return interleaveFields(t, a)

	case TypeVector2int16:
		return nil, errors.New("not implemented")

	case TypeCFrame:
		// The bytes of each value can vary in length.
		p := make([]Value, len(a))
		for i, cf := range a {
			cf := cf.(*ValueCFrame)
			// Build matrix part.
			b = append(b, cf.Special)
			if cf.Special == 0 {
				// Write all components.
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
		pb, _ := interleaveFields(TypeVector3, p)
		b = append(b, pb...)
		return b, nil

	case TypeCFrameQuat:
		p := make([]Value, len(a))
		for i, cf := range a {
			cf := cf.(*ValueCFrameQuat)
			b = append(b, cf.Special)
			if cf.Special == 0 {
				r := make([]byte, cf.quatBytesLen())
				cf.quatBytes(r)
				b = append(b, r...)
			}
			p[i] = &cf.Position
		}
		pb, _ := interleaveFields(TypeVector3, p)
		b = append(b, pb...)
		return b, nil

	case TypeToken:
		return interleaveAppend(t, a)

	case TypeReference:
		// Because values are generated in sequence, they are likely to be
		// relatively close to each other. Subtracting each value from the
		// previous will likely produce small values that compress well.
		if len(a) == 0 {
			return b, nil
		}
		const size = 4
		b = make([]byte, len(a)*size)
		var prev ValueReference
		for i, ref := range a {
			ref := ref.(*ValueReference)
			if i == 0 {
				ref.Bytes(b[i*size : i*size+size])
			} else {
				// Convert absolute ref to relative ref.
				(*ref - prev).Bytes(b[i*size : i*size+size])
			}
			prev = *ref
		}
		if err = interleave(b, size); err != nil {
			return nil, err
		}
		return b, nil

	case TypeVector3int16:
		return appendValueBytes(t, a), nil

	case TypeNumberSequence:
		return appendValueBytes(t, a), nil

	case TypeColorSequence:
		return appendValueBytes(t, a), nil

	case TypeNumberRange:
		return appendValueBytes(t, a), nil

	case TypeRect2D:
		return interleaveFields(t, a)

	case TypePhysicalProperties:
		// The bytes of each value can vary in length.
		q := make([]byte, 20)
		for _, pp := range a {
			pp := pp.(*ValuePhysicalProperties)
			b = append(b, pp.CustomPhysics)
			if pp.CustomPhysics != 0 {
				// Write all fields.
				binary.LittleEndian.PutUint32(q[0*4:0*4+4], math.Float32bits(pp.Density))
				binary.LittleEndian.PutUint32(q[1*4:1*4+4], math.Float32bits(pp.Friction))
				binary.LittleEndian.PutUint32(q[2*4:2*4+4], math.Float32bits(pp.Elasticity))
				binary.LittleEndian.PutUint32(q[3*4:3*4+4], math.Float32bits(pp.FrictionWeight))
				binary.LittleEndian.PutUint32(q[4*4:4*4+4], math.Float32bits(pp.ElasticityWeight))
				b = append(b, q...)
			}
		}
		return b, nil

	case TypeColor3uint8:
		return interleaveFields(t, a)

	case TypeInt64:
		return interleaveAppend(t, a)

	case TypeSharedString:
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
func deinterleaveFields(id Type, b []byte) (a []Value, err error) {
	if len(b) == 0 {
		return a, nil
	}

	if !id.Valid() {
		return nil, fmt.Errorf("type identifier 0x%X is not a valid Type.", id)
	}

	// Number of bytes per field
	nbytes := NewValue(id).(fielder).fieldLen()
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

	a = make([]Value, nvalues)

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
		v := NewValue(id)
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
func appendByteValues(id Type, b []byte) (a []Value, err error) {
	if size := id.Size(); size >= 0 {
		for i := 0; i+size <= len(b); i += size {
			v := NewValue(id)
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
		if len(ba) < 4 {
			return nil, errors.New("expected 4 more bytes in array")
		}
		size := int(binary.LittleEndian.Uint32(ba))
		if len(ba[4:]) < size*field {
			return nil, fmt.Errorf("expected %d more bytes in array", size*field)
		}

		v := NewValue(id)
		if err := v.FromBytes(ba[:4+size*field]); err != nil {
			return nil, err
		}
		a = append(a, v)

		ba = ba[4+size*field:]
	}
	return a, nil
}

// Deinterleave, then append from given size.
func deinterleaveAppend(t Type, b []byte) (a []Value, err error) {
	bc := make([]byte, len(b))
	copy(bc, b)
	size := t.Size()
	if err = deinterleave(bc, size); err != nil {
		return nil, err
	}
	return appendByteValues(t, bc)
}

// ValuesFromBytes decodes b according to t, into a slice of values, the type of
// each corresponding to t.
func ValuesFromBytes(t Type, b []byte) (a []Value, err error) {
	if !t.Valid() {
		return nil, fmt.Errorf("invalid type (%02X)", t)
	}

	switch t {
	case TypeString:
		return appendByteValues(t, b)

	case TypeBool:
		return appendByteValues(t, b)

	case TypeInt:
		return deinterleaveAppend(t, b)

	case TypeFloat:
		return deinterleaveAppend(t, b)

	case TypeDouble:
		return appendByteValues(t, b)

	case TypeUDim:
		return deinterleaveFields(t, b)

	case TypeUDim2:
		return deinterleaveFields(t, b)

	case TypeRay:
		return appendByteValues(t, b)

	case TypeFaces:
		return appendByteValues(t, b)

	case TypeAxes:
		return appendByteValues(t, b)

	case TypeBrickColor:
		return deinterleaveAppend(t, b)

	case TypeColor3:
		return deinterleaveFields(t, b)

	case TypeVector2:
		return deinterleaveFields(t, b)

	case TypeVector3:
		return deinterleaveFields(t, b)

	case TypeVector2int16:
		return nil, errors.New("not implemented")

	case TypeCFrame:
		cfs := make([]*ValueCFrame, 0)
		// This loop reads the matrix data. i is the current position in the
		// byte array. n is the expected size of the position data, which
		// increases every time another CFrame is read. As long as the number of
		// remaining bytes is greater than n, then the next byte can be assumed
		// to be matrix data.
		i := 0
		n := 0
		for ; len(b)-i > n; n += 12 {
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
		// ValueVector3.
		if a, err = deinterleaveFields(TypeVector3, b[i:i+n]); err != nil {
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
		return a, nil

	case TypeCFrameQuat:
		cfs := make([]*ValueCFrame, 0)
		i := 0
		n := 0
		for ; len(b)-i > n; n += 12 {
			cf := new(ValueCFrameQuat)
			cf.Special = b[i]
			i++
			if cf.Special == 0 {
				var q = cf.quatBytesLen()
				r := b[i:]
				if len(r) < q {
					return nil, fmt.Errorf("expected %d more bytes in array", q)
				}
				cf.quatFromBytes(r)
				i += q
			}
			c := cf.ToCFrame()
			cfs = append(cfs, &c)
		}
		if a, err = deinterleaveFields(TypeVector3, b[i:i+n]); err != nil {
			return nil, err
		}
		if len(a) != len(cfs) {
			return nil, errors.New("number of positions does not match number of matrices")
		}
		for i, p := range a {
			cfs[i].Position = *p.(*ValueVector3)
			a[i] = cfs[i]
		}
		return a, nil

	case TypeToken:
		return deinterleaveAppend(t, b)

	case TypeReference:
		if len(b) == 0 {
			return a, nil
		}
		const size = 4
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

	case TypeVector3int16:
		return appendByteValues(t, b)

	case TypeNumberSequence:
		return appendByteValues(t, b)

	case TypeColorSequence:
		return appendByteValues(t, b)

	case TypeNumberRange:
		return appendByteValues(t, b)

	case TypeRect2D:
		return deinterleaveFields(t, b)

	case TypePhysicalProperties:
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
		return a, nil

	case TypeColor3uint8:
		return deinterleaveFields(t, b)

	case TypeInt64:
		return deinterleaveAppend(t, b)

	case TypeSharedString:
		return deinterleaveAppend(t, b)

	default:
		return a, nil
	}
}
