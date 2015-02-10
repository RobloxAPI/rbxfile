package bin

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/bkaradzic/go-lz4"
	"io"
	"io/ioutil"
	"math"
)

////////////////////////////////////////////////////////////////

// BinaryHeader is the string indicating the start of a binary Roblox file.
const BinaryHeader = "<roblox!\x89\xff\r\n\x1a\n"

type ErrMismatchedVersion struct {
	ExpectedVersion uint16
	DecodedVersion  uint16
}

func (err ErrMismatchedVersion) Error() string {
	return fmt.Sprintf("expected version %d, decoded version is %d", err.ExpectedVersion, err.DecodedVersion)
}

var (
	ErrCorruptHeader = errors.New("the file header is corrupted")
)

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

// Returns the size of an integer.
func intDataSize(data interface{}) int {
	switch data.(type) {
	case int8, *int8, *uint8:
		return 1
	case int16, *int16, *uint16:
		return 2
	case int32, *int32, *uint32:
		return 4
	case int64, *int64, *uint64:
		return 8
	}
	return 0
}

// Reads a number from a Reader while keeping track of the number of bytes
// read.
func readNumber(r io.Reader, order binary.ByteOrder, data interface{}, n *int64) (err error) {
	if m := intDataSize(data); m != 0 {
		var b [8]byte
		bs := b[:m]

		nn, err := io.ReadFull(r, bs)
		*n += int64(nn)
		if err != nil {
			return err
		}

		switch data := data.(type) {
		case *int8:
			*data = int8(b[0])
		case *uint8:
			*data = b[0]
		case *int16:
			*data = int16(order.Uint16(bs))
		case *uint16:
			*data = order.Uint16(bs)
		case *int32:
			*data = int32(order.Uint32(bs))
		case *uint32:
			*data = order.Uint32(bs)
		case *int64:
			*data = int64(order.Uint64(bs))
		case *uint64:
			*data = order.Uint64(bs)
		default:
			goto invalid
		}

		return nil
	}

invalid:
	panic("invalid type")
}

// Returns a string read from a Reader while keeping track of the number of
// bytes.
func readString(r io.Reader, n *int64) (str string, err error) {
	var length uint32
	if err = readNumber(r, binary.LittleEndian, &length, n); err != nil {
		return "", err
	}

	s := make([]byte, length)
	nn, err := io.ReadFull(r, s)
	*n += int64(nn)
	if err != nil {
		return "", err
	}

	return string(s), nil
}

////////////////////////////////////////////////////////////////

// Warning is a non-fatal message emitted by the decoder.
type Warning interface {
	Warn() string
}

func warning(text string) Warning {
	return &warningString{text}
}

type warningString struct {
	s string
}

func (e *warningString) Warn() string {
	return e.s
}

////////////////////////////////////////////////////////////////

// ChunkGenerator is a function that initializes a type which implements a
// Chunk.
type ChunkGenerator func() Chunk

// FormatModel models Roblox's binary file format. Directly, it can be used to
// control exactly how a file is encoded.
type FormatModel struct {
	// ChunkGenerators maps a chunk signature to a ChunkGenerator, which is
	// used by the decoder to look up what kind of chunks can be decoded.
	ChunkGenerators map[[4]byte]ChunkGenerator

	// GroupCount is the number of instance groups in the model.
	GroupCount uint32

	// InstanceCount is the number of unique instances in the model.
	InstanceCount uint32

	// Chunks is a list of Chunks present in the model.
	Chunks []Chunk

	// Warnings is a list of non-fatal problems that were encountered while
	// decoding.
	Warnings []Warning
}

// NewFormatModel returns a FormatModel initialized with the current version
// of the format codec.
func NewFormatModel() *FormatModel {
	f := new(FormatModel)
	f.ChunkGenerators = map[[4]byte]ChunkGenerator{
		newChunkInstance().Signature(): newChunkInstance,
		newChunkProperty().Signature(): newChunkProperty,
		newChunkParent().Signature():   newChunkParent,
		newChunkEnd().Signature():      newChunkEnd,
	}
	return f
}

// ReadFrom decodes data from r into the FormatModel.
func (f *FormatModel) ReadFrom(r io.Reader) (n int64, err error) {
	// reuse space from previous slices
	f.Warnings = f.Warnings[:0]
	f.Chunks = f.Chunks[:0]

	header := make([]byte, len(BinaryHeader))
	nn, err := io.ReadFull(r, header)
	n += int64(nn)
	if err != nil {
		return n, err
	}

	if !bytes.Equal(header, []byte(BinaryHeader)) {
		return n, ErrCorruptHeader
	}

	var version uint16
	if err = readNumber(r, binary.LittleEndian, &version, &n); err != nil {
		return n, err
	}
	if version != 0 {
		return n, ErrMismatchedVersion{ExpectedVersion: 0, DecodedVersion: version}
	}

	if err = readNumber(r, binary.LittleEndian, &f.GroupCount, &n); err != nil {
		return n, err
	}

	if err = readNumber(r, binary.LittleEndian, &f.InstanceCount, &n); err != nil {
		return n, err
	}

	var reserved uint64
	if err = readNumber(r, binary.LittleEndian, &reserved, &n); err != nil {
		return n, err
	}
	if reserved != 0 {
		f.Warnings = append(f.Warnings, warning("reserved space in file header is non-zero"))
	}

loop:
	for {
		data, n, err := decompressChunk(r)
		n += int64(nn)
		if err != nil {
			return n, err
		}

		newChunk, ok := f.ChunkGenerators[data.signature]
		if !ok {
			f.Warnings = append(f.Warnings, warning("unknown chunk signature `"+string(data.signature[:])+"`"))
			continue loop
		}

		chunk := newChunk()
		chunk.SetCompressed(data.compressedLength != 0)
		if _, err := chunk.ReadFrom(data.decompressedData); err != nil {
			return n, err
		}

		f.Chunks = append(f.Chunks, chunk)

		if endChunk, ok := chunk.(*ChunkEnd); ok {
			if !endChunk.isCompressed {
				f.Warnings = append(f.Warnings, warning("END chunk is not uncompressed"))
			}

			if !bytes.Equal(endChunk.Content, []byte("</roblox>")) {
				f.Warnings = append(f.Warnings, warning("END chunk content is not `</roblox>`"))
			}

			break loop
		}
	}

	return n, nil
}

// WriteTo encodes the FormatModel as bytes to w.
func (f *FormatModel) WriteTo(w io.Writer) (n int64, err error) {
	return 0, errors.New("not implemented")
}

////////////////////////////////////////////////////////////////

// Chunk is a portion of the model that contains distinct data.
type Chunk interface {
	// Signature returns a signature used to identify the chunk's type.
	Signature() [4]byte

	// Compressed returns whether the chunk was compressed when decoding, or
	// whether the chunk should be compressed when encoding.
	Compressed() bool

	// SetCompressed sets whether the chunk should be compressed when
	// encoding.
	SetCompressed(bool)

	// ReadFrom reads data from a stream into the chunk. Assumes the signature
	// has already been read.
	ReadFrom(r io.Reader) (n int64, err error)

	// WriteTo writes the data from a chunk to the stream. This includes the
	// signature.
	WriteTo(w io.Writer) (n int64, err error)
}

// Decompresses a chunk and prepares it for reading.
type compressedChunk struct {
	signature          [4]byte
	compressedLength   uint32
	decompressedLength uint32
	decompressedData   io.Reader
}

// Decompresses a lz4-compressed chunk and returns a reader that reads the
// decompressed data.
func decompressChunk(r io.Reader) (data *compressedChunk, n int64, err error) {
	sigb := data.signature[:]
	nn, err := io.ReadFull(r, sigb)
	n += int64(nn)
	if err != nil {
		return nil, n, err
	}

	if err = readNumber(r, binary.LittleEndian, &data.compressedLength, &n); err != nil {
		return nil, n, err
	}

	if err = readNumber(r, binary.LittleEndian, &data.decompressedLength, &n); err != nil {
		return nil, n, err
	}

	var reserved uint32
	if err = readNumber(r, binary.LittleEndian, &reserved, &n); err != nil {
		return nil, n, err
	}

	decompressedData := make([]byte, data.decompressedLength)
	// If compressed length is 0, then the data is not compressed.
	if data.compressedLength == 0 {
		nn, err := io.ReadFull(r, decompressedData)
		n += int64(nn)
		if err != nil {
			return nil, n, err
		}
	} else {
		// Prepare compressed data for reading by lz4, which requires the
		// uncompressed length before the compressed data.
		compressedData := make([]byte, data.compressedLength+4)
		binary.LittleEndian.PutUint32(compressedData, data.decompressedLength)

		nn, err := io.ReadFull(r, compressedData[4:])
		n += int64(nn)
		if err != nil {
			return nil, n, err
		}

		// ROBLOX ERROR: "Malformed data ([true decompressed length] != [given decompressed length])"
		// lz4 already does some kind of size validation, though the error message isn't the same.

		if _, err := lz4.Decode(decompressedData, compressedData); err != nil {
			return nil, n, err
		}
	}

	data.decompressedData = bytes.NewReader(decompressedData)
	return data, n, nil
}

////////////////////////////////////////////////////////////////

// ChunkInstance is a Chunk that contains information about the instances in
// the file. Instances of the same ClassName are grouped together into this
// kind of chunk, which are called "instance groups".
type ChunkInstance struct {
	// Whether the chunk is compressed.
	isCompressed bool

	// GroupID is a number identifying the instance group.
	GroupID uint32

	// ClassName indicates the ClassName property of each instance in the
	// group.
	ClassName string

	// InstanceIDs is a list of numbers that identify each instance in the
	// group, which can be referred to in other chunks. The length of the
	// array indicates how many instances are in the group.
	InstanceIDs []int32

	// IsService indicates the chunk has GetService flags.
	IsService bool

	// GetService is a list of flags indicating how to treat each instance in
	// the group. Each byte in the list corresponds to the instance in
	// InstanceIDs.
	//
	// A value of 0x0 will treat the instance normally, using Instance.new()
	// to create the instance.
	//
	// A value of 0x1 will treat the instance as a service, using
	// game:GetService() to get the instance.
	GetService []byte
}

func newChunkInstance() Chunk {
	return new(ChunkInstance)
}

func (ChunkInstance) Signature() [4]byte {
	return [4]byte{0x49, 0x4E, 0x53, 0x54} // INST
}

func (c *ChunkInstance) Compressed() bool {
	return c.isCompressed
}

func (c *ChunkInstance) SetCompressed(b bool) {
	c.isCompressed = b
}

func (c *ChunkInstance) ReadFrom(r io.Reader) (n int64, err error) {
	if err = readNumber(r, binary.LittleEndian, &c.GroupID, &n); err != nil {
		return n, err
	}

	if c.ClassName, err = readString(r, &n); err != nil {
		return n, err
	}

	var isService uint8
	if err = readNumber(r, binary.LittleEndian, &isService, &n); err != nil {
		return n, err
	}
	c.IsService = isService != 0

	var groupLength uint32
	if err = readNumber(r, binary.LittleEndian, &groupLength, &n); err != nil {
		return n, err
	}

	groupRaw := make([]byte, groupLength*4)
	nn, err := io.ReadFull(r, groupRaw)
	n += int64(nn)
	if err != nil {
		return n, err
	}

	deinterleave(groupRaw, 4)

	c.InstanceIDs = make([]int32, groupLength)
	if groupLength > 0 {
		c.InstanceIDs[0] = decodeZigzag(binary.BigEndian.Uint32(groupRaw[0:4]))
		for i := uint32(1); i < groupLength; i++ {
			// Each entry is relative to the previous
			c.InstanceIDs[i] = c.InstanceIDs[i-1] + decodeZigzag(binary.BigEndian.Uint32(groupRaw[i*4:i*4+4]))
		}
	}

	if c.IsService {
		c.GetService = make([]byte, groupLength)
		nn, err := io.ReadFull(r, c.GetService)
		n += int64(nn)
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func (c *ChunkInstance) WriteTo(w io.Writer) (n int64, err error) {
	return 0, errors.New("not implemented")
}

////////////////////////////////////////////////////////////////

// ChunkEnd is a Chunk that signals the end of the file. It causes the decoder
// to stop reading chunks, so it should be the last chunk.
type ChunkEnd struct {
	// Whether the chunk is compressed.
	isCompressed bool

	// The raw decompressed content of the chunk. For maximum compatibility,
	// the content should be "</roblox>", and the chunk should be
	// uncompressed. The decoder will emit warnings indicating such, if this
	// is not the case.
	Content []byte
}

func newChunkEnd() Chunk {
	return new(ChunkEnd)
}

func (ChunkEnd) Signature() [4]byte {
	return [4]byte{0x45, 0x4E, 0x44, 0x00} // END\0
}

func (c *ChunkEnd) Compressed() bool {
	return c.isCompressed
}

func (c *ChunkEnd) SetCompressed(b bool) {
	c.isCompressed = b
}

func (c *ChunkEnd) ReadFrom(r io.Reader) (n int64, err error) {
	c.Content, err = ioutil.ReadAll(r)
	n += int64(len(c.Content))
	if err != nil {
		return n, err
	}

	return n, nil
}

func (c *ChunkEnd) WriteTo(w io.Writer) (n int64, err error) {
	return 0, errors.New("not implemented")
}

////////////////////////////////////////////////////////////////

// ChunkParent is a Chunk that contains information about the parent-child
// relationships between instances in the model.
type ChunkParent struct {
	// Whether the chunk is compressed.
	isCompressed bool

	// Version is the version of the chunk. Reserved so that the format of the
	// parent chunk can be changed without changing the version of the entire
	// file format.
	Version uint8

	// InstanceCount is the number of instances that are described in the
	// chunk.
	InstanceCount uint32

	// Children is a list of instances referred to by instance ID. The length
	// of this array should be equal to InstanceCount.
	Children []int32

	// Parents is a list of instances, referred to by instance ID, that
	// indicate the Parent of the corresponding instance in the Children
	// array. The length of this array should be equal to InstanceCount.
	Parents []int32
}

func newChunkParent() Chunk {
	return new(ChunkParent)
}

func (ChunkParent) Signature() [4]byte {
	return [4]byte{0x50, 0x52, 0x4E, 0x54} // PRNT
}

func (c *ChunkParent) Compressed() bool {
	return c.isCompressed
}

func (c *ChunkParent) SetCompressed(b bool) {
	c.isCompressed = b
}

func (c *ChunkParent) ReadFrom(r io.Reader) (n int64, err error) {
	if err = readNumber(r, binary.LittleEndian, &c.Version, &n); err != nil {
		return n, err
	}

	if err = readNumber(r, binary.LittleEndian, &c.InstanceCount, &n); err != nil {
		return n, err
	}

	childrenRaw := make([]byte, c.InstanceCount*4)
	nn, err := io.ReadFull(r, childrenRaw)
	n += int64(nn)
	if err != nil {
		return n, err
	}

	deinterleave(childrenRaw, 4)

	c.Children = make([]int32, c.InstanceCount)
	if c.InstanceCount > 0 {
		c.Children[0] = decodeZigzag(binary.BigEndian.Uint32(childrenRaw[0:4]))
		for i := uint32(1); i < c.InstanceCount; i++ {
			// Each entry is relative to the previous
			c.Children[i] = c.Children[i-1] + decodeZigzag(binary.BigEndian.Uint32(childrenRaw[i*4:i*4+4]))
		}
	}

	parentsRaw := make([]byte, c.InstanceCount*4)
	nn, err = io.ReadFull(r, parentsRaw)
	n += int64(nn)
	if err != nil {
		return n, err
	}

	deinterleave(parentsRaw, 4)

	c.Parents = make([]int32, c.InstanceCount)
	if c.InstanceCount > 0 {
		c.Parents[0] = decodeZigzag(binary.BigEndian.Uint32(parentsRaw[0:4]))
		for i := uint32(1); i < c.InstanceCount; i++ {
			// Each entry is relative to the previous
			c.Parents[i] = c.Parents[i-1] + decodeZigzag(binary.BigEndian.Uint32(parentsRaw[i*4:i*4+4]))
		}
	}

	return n, nil
}

func (c *ChunkParent) WriteTo(w io.Writer) (n int64, err error) {
	return 0, errors.New("not implemented")
}

////////////////////////////////////////////////////////////////

// ChunkProperty is a Chunk that contains information about the properties of
// a group of instances.
type ChunkProperty struct {
	// Whether the chunk is compressed.
	isCompressed bool

	// GroupID is the ID of an instance group contained in a ChunkInstance.
	GroupID uint32

	// PropertyName is the name of a valid property in each instance of the
	// corresponding instance group.
	PropertyName string

	// DataType is a number indicating the type of the property. It
	// corresponds to the result of the Value.TypeID method.
	DataType uint8

	// Properties is a list of Values of the given DataType. Each value in the
	// array corresponds to the property of an instance in the specified
	// group.
	Properties []Value
}

func newChunkProperty() Chunk {
	return new(ChunkProperty)
}

func (ChunkProperty) Signature() [4]byte {
	return [4]byte{0x50, 0x52, 0x4F, 0x50} // PROP
}

func (c *ChunkProperty) Compressed() bool {
	return c.isCompressed
}

func (c *ChunkProperty) SetCompressed(b bool) {
	c.isCompressed = b
}

func (c *ChunkProperty) ReadFrom(r io.Reader) (n int64, err error) {
	if err = readNumber(r, binary.LittleEndian, &c.GroupID, &n); err != nil {
		return n, err
	}

	if c.PropertyName, err = readString(r, &n); err != nil {
		return n, err
	}

	if err = readNumber(r, binary.LittleEndian, &c.DataType, &n); err != nil {
		return n, err
	}

	rawBytes, err := ioutil.ReadAll(r)
	n += int64(len(rawBytes))
	if err != nil {
		return n, nil
	}

	newValue, ok := valueGenerators[c.DataType]
	if !ok {
		return n, errors.New("unrecognized data type")
	}

	c.Properties, err = newValue().FromArrayBytes(rawBytes)
	if err != nil {
		return n, err
	}

	return n, nil
}

func (c *ChunkProperty) WriteTo(w io.Writer) (n int64, err error) {
	return 0, errors.New("not implemented")
}

////////////////////////////////////////////////////////////////
