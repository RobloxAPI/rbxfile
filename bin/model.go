package bin

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/bkaradzic/go-lz4"
	"io"
	"io/ioutil"
)

////////////////////////////////////////////////////////////////

// RobloxSig is the signature a Roblox file (binary or XML).
const RobloxSig = "<roblox"

// BinaryMarker indicates the start of a binary file, rather than an XML file.
const BinaryMarker = "!"

// BinaryHeader is the header magic of a binary file.
const BinaryHeader = "\x89\xff\r\n\x1a\n"

var (
	ErrInvalidSig       = errors.New("invalid signature")
	ErrCorruptHeader    = errors.New("the file header is corrupted")
	ErrChunkParentArray = errors.New("length of parent array does not match children array")
)

type ErrUnrecognizedVersion uint16

func (err ErrUnrecognizedVersion) Error() string {
	return fmt.Sprintf("unrecognized version %d", err)
}

// ErrChunk is an error produced by a chunk of a certain type.
type ErrChunk struct {
	Sig [4]byte
	Err error
}

func (err ErrChunk) Error() string {
	return fmt.Sprintf("chunk %s: %s", err.Sig, err.Err.Error())
}

type ErrInvalidType struct {
	Chunk *ChunkProperty
	Bytes []byte
}

func (err *ErrInvalidType) Error() string {
	return fmt.Sprintf("invalid data type 0x%X", byte(err.Chunk.DataType))
}

// ErrValue is an error that is produced by a Value of a certain Type.
type ErrValue struct {
	Type  Type
	Bytes []byte
	Err   error
}

func (err ErrValue) Error() string {
	return fmt.Sprintf("type %s (0x%X): %s", err.Type.String(), byte(err.Type), err.Err.Error())
}

var (
	WarnReserveNonZero     = errors.New("reserved space in file header is non-zero")
	WarnEndChunkCompressed = errors.New("end chunk is compressed")
	WarnEndChunkContent    = errors.New("end chunk content is not `</roblox>`")
	WarnEndChunkNotLast    = errors.New("end chunk is not the last chunk")
)

type WarnUnknownChunk [4]byte

func (w WarnUnknownChunk) Error() string {
	return fmt.Sprintf("unknown chunk signature `%s`", [4]byte(w))
}

////////////////////////////////////////////////////////////////

// Returns the size of an integer.
func intDataSize(data interface{}) int {
	switch data.(type) {
	case int8, *int8, uint8, *uint8:
		return 1
	case int16, *int16, uint16, *uint16:
		return 2
	case int32, *int32, uint32, *uint32:
		return 4
	case int64, *int64, uint64, *uint64:
		return 8
	}
	return 0
}

// Reader wrapper that keeps track of the number of bytes written.
type formatReader struct {
	r   io.Reader
	n   int64
	err error
}

func (f *formatReader) read(p []byte) (failed bool) {
	if f.err != nil {
		return true
	}

	var n int
	n, f.err = io.ReadFull(f.r, p)
	f.n += int64(n)

	if f.err != nil {
		return true
	}

	return false
}

func (f *formatReader) readall() (data []byte, failed bool) {
	if f.err != nil {
		return nil, true
	}

	data, f.err = ioutil.ReadAll(f.r)
	f.n += int64(len(data))

	if f.err != nil {
		return nil, true
	}

	return data, false
}

func (f *formatReader) end() (n int64, err error) {
	return f.n, f.err
}

func (f *formatReader) readNumber(order binary.ByteOrder, data interface{}) (failed bool) {
	if f.err != nil {
		return true
	}

	if m := intDataSize(data); m != 0 {
		var b [8]byte
		bs := b[:m]

		if f.read(bs) {
			return true
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

		return false
	}

invalid:
	panic("invalid type")
}

func (f *formatReader) readString(data *string) (failed bool) {
	if f.err != nil {
		return true
	}

	var length uint32
	if f.readNumber(binary.LittleEndian, &length) {
		return true
	}

	s := make([]byte, length)
	if f.read(s) {
		return true
	}

	*data = string(s)

	return false
}

// Writer wrapper that keeps track of the number of bytes written.
type formatWriter struct {
	w   io.Writer
	n   int64
	err error
}

func (f *formatWriter) write(p []byte) (failed bool) {
	if f.err != nil {
		return true
	}

	var n int
	n, f.err = f.w.Write(p)
	f.n += int64(n)

	if n < len(p) {
		return true
	}

	return false
}

func (f *formatWriter) end() (n int64, err error) {
	return f.n, f.err
}

func (f *formatWriter) writeNumber(order binary.ByteOrder, data interface{}) (failed bool) {
	if f.err != nil {
		return true
	}

	if m := intDataSize(data); m != 0 {
		b := make([]byte, 8)

		switch data := data.(type) {
		case int8:
			b[0] = uint8(data)
		case uint8:
			b[0] = data
		case int16:
			order.PutUint16(b, uint16(data))
		case uint16:
			order.PutUint16(b, data)
		case int32:
			order.PutUint32(b, uint32(data))
		case uint32:
			order.PutUint32(b, data)
		case int64:
			order.PutUint64(b, uint64(data))
		case uint64:
			order.PutUint64(b, data)
		default:
			goto invalid
		}

		return f.write(b[:m])
	}

invalid:
	panic("invalid type")
}

func (f *formatWriter) writeString(data string) (failed bool) {
	if f.err != nil {
		return true
	}

	if f.writeNumber(binary.LittleEndian, uint32(len(data))) {
		return true
	}

	return f.write([]byte(data))
}

////////////////////////////////////////////////////////////////

// chunkGenerator is a function that initializes a type which implements a
// Chunk.
type chunkGenerator func() Chunk

// chunkGenerators returns a function that generates a chunk of the given
// signature, which exists for the given format version.
func chunkGenerators(version uint16, sig [4]byte) chunkGenerator {
	switch version {
	case 0:
		switch sig {
		case newChunkInstance().Signature():
			return newChunkInstance
		case newChunkProperty().Signature():
			return newChunkProperty
		case newChunkParent().Signature():
			return newChunkParent
		case newChunkMeta().Signature():
			return newChunkMeta
		case newChunkEnd().Signature():
			return newChunkEnd
		default:
			return nil
		}
	default:
		return nil
	}
}

// validChunk returns whether a chunk signature is valid for a format version.
func validChunk(version uint16, sig [4]byte) bool {
	return chunkGenerators(version, sig) != nil
}

// FormatModel models Roblox's binary file format. Directly, it can be used to
// control exactly how a file is encoded.
type FormatModel struct {
	// Version indicates the version of the format model.
	Version uint16

	// TypeCount is the number of instance types in the model.
	TypeCount uint32

	// InstanceCount is the number of unique instances in the model.
	InstanceCount uint32

	// Chunks is a list of Chunks present in the model.
	Chunks []Chunk

	// If Strict is true, certain errors normally emitted as warnings are
	// instead emitted as errors.
	Strict bool

	// Warnings is a list of non-fatal problems that have occurred. This will
	// be cleared and populated when calling either ReadFrom and WriteTo.
	// Codecs may also clear and populate this when decoding or encoding.
	Warnings []error
}

// ReadFrom decodes data from r into the FormatModel.
//
// If an error occurs while reading a chunk, the error is emitted as a
// ErrChunk to FormatModel.Warnings, unless FormatModel.Strict is true.
func (f *FormatModel) ReadFrom(r io.Reader) (n int64, err error) {
	if r == nil {
		return 0, errors.New("reader is nil")
	}

	fr := &formatReader{r: r}

	sig := make([]byte, len(RobloxSig+BinaryMarker))
	if fr.read(sig) {
		return fr.end()
	}

	if !bytes.Equal(sig, []byte(RobloxSig+BinaryMarker)) {
		fr.err = ErrInvalidSig
		return fr.end()
	}

	header := make([]byte, len(BinaryHeader))
	if fr.read(header) {
		return fr.end()
	}

	if !bytes.Equal(header, []byte(BinaryHeader)) {
		fr.err = ErrCorruptHeader
		return fr.end()
	}

	var version uint16
	if fr.readNumber(binary.LittleEndian, &version) {
		return fr.end()
	}

	switch version {
	default:
		fr.err = ErrUnrecognizedVersion(version)
		return fr.end()
	case 0:
	}

	f.Version = version

	// reuse space from previous slices
	f.Warnings = f.Warnings[:0]
	f.Chunks = f.Chunks[:0]

	if fr.readNumber(binary.LittleEndian, &f.TypeCount) {
		return fr.end()
	}

	if fr.readNumber(binary.LittleEndian, &f.InstanceCount) {
		return fr.end()
	}

	var reserved uint64
	if fr.readNumber(binary.LittleEndian, &reserved) {
		return fr.end()
	}
	if reserved != 0 {
		f.Warnings = append(f.Warnings, WarnReserveNonZero)
	}

loop:
	for {
		rawChunk := new(rawChunk)
		if rawChunk.ReadFrom(fr) {
			return fr.end()
		}

		newChunk := chunkGenerators(f.Version, rawChunk.signature)
		if newChunk == nil {
			f.Warnings = append(f.Warnings, WarnUnknownChunk(rawChunk.signature))
			continue loop
		}

		chunk := newChunk()
		chunk.SetCompressed(rawChunk.compressed)

		if _, err := chunk.ReadFrom(bytes.NewReader(rawChunk.payload)); err != nil {
			err = ErrChunk{Sig: rawChunk.signature, Err: err}
			if f.Strict {
				fr.err = err
				return fr.end()
			}
			f.Warnings = append(f.Warnings, err)
			continue loop
		}

		f.Chunks = append(f.Chunks, chunk)

		if endChunk, ok := chunk.(*ChunkEnd); ok {
			if endChunk.Compressed() {
				f.Warnings = append(f.Warnings, WarnEndChunkCompressed)
			}

			if !bytes.Equal(endChunk.Content, []byte("</roblox>")) {
				f.Warnings = append(f.Warnings, WarnEndChunkContent)
			}

			break loop
		}
	}

	return fr.end()
}

// WriteTo encodes the FormatModel as bytes to w.
func (f *FormatModel) WriteTo(w io.Writer) (n int64, err error) {
	if w == nil {
		return 0, errors.New("writer is nil")
	}

	f.Warnings = f.Warnings[:0]

	fw := &formatWriter{w: w}

	if fw.write([]byte(RobloxSig + BinaryMarker + BinaryHeader)) {
		return fw.end()
	}

	if fw.writeNumber(binary.LittleEndian, f.Version) {
		return fw.end()
	}

	if fw.writeNumber(binary.LittleEndian, f.TypeCount) {
		return fw.end()
	}

	if fw.writeNumber(binary.LittleEndian, f.InstanceCount) {
		return fw.end()
	}

	// reserved
	if fw.writeNumber(binary.LittleEndian, uint64(0)) {
		return fw.end()
	}

	for i, chunk := range f.Chunks {
		if !validChunk(f.Version, chunk.Signature()) {
			f.Warnings = append(f.Warnings, WarnUnknownChunk(chunk.Signature()))
			continue
		}

		if endChunk, ok := chunk.(*ChunkEnd); ok {
			if endChunk.IsCompressed {
				f.Warnings = append(f.Warnings, WarnEndChunkCompressed)
			}

			if !bytes.Equal(endChunk.Content, []byte("</roblox>")) {
				f.Warnings = append(f.Warnings, WarnEndChunkContent)
			}

			if i != len(f.Chunks)-1 {
				f.Warnings = append(f.Warnings, WarnEndChunkNotLast)
			}
		}

		rawChunk := new(rawChunk)
		rawChunk.signature = chunk.Signature()
		rawChunk.compressed = chunk.Compressed()

		buf := new(bytes.Buffer)
		if _, fw.err = chunk.WriteTo(buf); fw.err != nil {
			return fw.end()
		}

		rawChunk.payload = buf.Bytes()

		if rawChunk.WriteTo(fw) {
			return fw.end()
		}
	}

	return fw.end()
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

	// ReadFrom processes the payload of a decompressed chunk.
	ReadFrom(r io.Reader) (n int64, err error)

	// WriteTo writes the data from a chunk to an uncompressed payload. The
	// payload will be compressed afterward depending on the chunk's
	// compression settings.
	WriteTo(w io.Writer) (n int64, err error)
}

// Represents a raw chunk, which contains compression data and payload.
type rawChunk struct {
	signature  [4]byte
	compressed bool
	payload    []byte
}

// Reads out a raw chunk from a stream, decompressing the chunk if necessary.
func (c *rawChunk) ReadFrom(fr *formatReader) bool {
	if fr.read(c.signature[:]) {
		return true
	}

	var compressedLength uint32
	if fr.readNumber(binary.LittleEndian, &compressedLength) {
		return true
	}

	var decompressedLength uint32
	if fr.readNumber(binary.LittleEndian, &decompressedLength) {
		return true
	}

	var reserved uint32
	if fr.readNumber(binary.LittleEndian, &reserved) {
		return true
	}

	c.payload = make([]byte, decompressedLength)
	// If compressed length is 0, then the data is not compressed.
	if compressedLength == 0 {
		c.compressed = false
		if fr.read(c.payload) {
			return true
		}
	} else {
		c.compressed = true

		// Prepare compressed data for reading by lz4, which requires the
		// uncompressed length before the compressed data.
		compressedData := make([]byte, compressedLength+4)
		binary.LittleEndian.PutUint32(compressedData, decompressedLength)

		if fr.read(compressedData[4:]) {
			return true
		}

		// ROBLOX ERROR: "Malformed data ([true decompressed length] != [given
		// decompressed length])". lz4 already does some kind of size
		// validation, though the error message isn't the same.

		if _, err := lz4.Decode(c.payload, compressedData); err != nil {
			fr.err = fmt.Errorf("lz4: %s", err.Error())
			return true
		}
	}

	return false
}

// Writes a raw chunk payload to a stream, compressing if necessary.
func (c *rawChunk) WriteTo(fw *formatWriter) bool {
	if fw.write(c.signature[:]) {
		return true
	}

	if c.compressed {
		var compressedData []byte
		compressedData, fw.err = lz4.Encode(compressedData, c.payload)
		if fw.err != nil {
			return true
		}

		// lz4 sanity check
		if binary.LittleEndian.Uint32(compressedData[:4]) != uint32(len(c.payload)) {
			panic("lz4 uncompressed length does not match payload length")
		}

		// Compressed length; lz4 prepends the length of the uncompressed
		// payload, so it must be excluded.
		compressedPayload := compressedData[4:]

		if fw.writeNumber(binary.LittleEndian, uint32(len(compressedPayload))) {
			return true
		}

		// Decompressed length
		if fw.writeNumber(binary.LittleEndian, uint32(len(c.payload))) {
			return true
		}

		// Reserved
		if fw.writeNumber(binary.LittleEndian, uint32(0)) {
			return true
		}

		if fw.write(compressedPayload) {
			return true
		}
	} else {
		// If the data is not compressed, then the compressed length is 0
		if fw.writeNumber(binary.LittleEndian, uint32(0)) {
			return true
		}

		// Decompressed length
		if fw.writeNumber(binary.LittleEndian, uint32(len(c.payload))) {
			return true
		}

		// Reserved
		if fw.writeNumber(binary.LittleEndian, uint32(0)) {
			return true
		}

		if fw.write(c.payload) {
			return true
		}
	}

	return false
}

////////////////////////////////////////////////////////////////

// ChunkInstance is a Chunk that contains information about the instances in
// the file. Instances of the same ClassName are grouped together into this
// kind of chunk, which are called "instance groups".
type ChunkInstance struct {
	// Whether the chunk is compressed.
	IsCompressed bool

	// TypeID is a number identifying the instance group.
	TypeID int32

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
	return c.IsCompressed
}

func (c *ChunkInstance) SetCompressed(b bool) {
	c.IsCompressed = b
}

func (c *ChunkInstance) ReadFrom(r io.Reader) (n int64, err error) {
	fr := &formatReader{r: r}

	if fr.readNumber(binary.LittleEndian, &c.TypeID) {
		return fr.end()
	}

	if fr.readString(&c.ClassName) {
		return fr.end()
	}

	var isService uint8
	if fr.readNumber(binary.LittleEndian, &isService) {
		return fr.end()
	}
	c.IsService = isService != 0

	var groupLength uint32
	if fr.readNumber(binary.LittleEndian, &groupLength) {
		return fr.end()
	}

	c.InstanceIDs = make([]int32, groupLength)
	if groupLength > 0 {
		raw := make([]byte, groupLength*4)
		if fr.read(raw) {
			return fr.end()
		}

		var values []Value
		if values, fr.err = ValueReference(0).FromArrayBytes(raw); fr.err != nil {
			return fr.end()
		}

		for i, v := range values {
			c.InstanceIDs[i] = int32(*v.(*ValueReference))
		}
	}

	if c.IsService {
		c.GetService = make([]byte, groupLength)
		if fr.read(c.GetService) {
			return fr.end()
		}
	}

	return fr.end()
}

func (c *ChunkInstance) WriteTo(w io.Writer) (n int64, err error) {
	fw := &formatWriter{w: w}

	if fw.writeNumber(binary.LittleEndian, c.TypeID) {
		return fw.end()
	}

	if fw.writeString(c.ClassName) {
		return fw.end()
	}

	var isService uint8 = 0
	if c.IsService {
		isService = 1
	}
	if fw.writeNumber(binary.LittleEndian, isService) {
		return fw.end()
	}

	if fw.writeNumber(binary.LittleEndian, uint32(len(c.InstanceIDs))) {
		return fw.end()
	}

	if len(c.InstanceIDs) > 0 {
		values := make([]Value, len(c.InstanceIDs))
		for i, id := range c.InstanceIDs {
			n := id
			values[i] = (*ValueReference)(&n)
		}

		var raw []byte
		if raw, fw.err = new(ValueReference).ArrayBytes(values); fw.err != nil {
			return fw.end()
		}

		if fw.write(raw) {
			return fw.end()
		}
	}

	if c.IsService {
		if fw.write(c.GetService) {
			return fw.end()
		}
	}

	return fw.end()
}

////////////////////////////////////////////////////////////////

// ChunkEnd is a Chunk that signals the end of the file. It causes the decoder
// to stop reading chunks, so it should be the last chunk.
type ChunkEnd struct {
	// Whether the chunk is compressed.
	IsCompressed bool

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
	return c.IsCompressed
}

func (c *ChunkEnd) SetCompressed(b bool) {
	c.IsCompressed = b
}

func (c *ChunkEnd) ReadFrom(r io.Reader) (n int64, err error) {
	fr := &formatReader{r: r}

	c.Content, _ = fr.readall()

	return fr.end()
}

func (c *ChunkEnd) WriteTo(w io.Writer) (n int64, err error) {
	fw := &formatWriter{w: w}

	fw.write(c.Content)

	return fw.end()
}

////////////////////////////////////////////////////////////////

// ChunkParent is a Chunk that contains information about the parent-child
// relationships between instances in the model.
type ChunkParent struct {
	// Whether the chunk is compressed.
	IsCompressed bool

	// Version is the version of the chunk. Reserved so that the format of the
	// parent chunk can be changed without changing the version of the entire
	// file format.
	Version uint8

	// Children is a list of instances referred to by instance ID. The length
	// of this array should be equal to InstanceCount.
	Children []int32

	// Parents is a list of instances, referred to by instance ID, that
	// indicate the Parent of the corresponding instance in the Children
	// array. The length of this array should be equal to the length of
	// Children.
	Parents []int32
}

func newChunkParent() Chunk {
	return new(ChunkParent)
}

func (ChunkParent) Signature() [4]byte {
	return [4]byte{0x50, 0x52, 0x4E, 0x54} // PRNT
}

func (c *ChunkParent) Compressed() bool {
	return c.IsCompressed
}

func (c *ChunkParent) SetCompressed(b bool) {
	c.IsCompressed = b
}

func (c *ChunkParent) ReadFrom(r io.Reader) (n int64, err error) {
	fr := &formatReader{r: r}

	if fr.readNumber(binary.LittleEndian, &c.Version) {
		return fr.end()
	}

	var instanceCount uint32
	if fr.readNumber(binary.LittleEndian, &instanceCount) {
		return fr.end()
	}

	c.Children = make([]int32, instanceCount)
	if instanceCount > 0 {
		raw := make([]byte, instanceCount*4)
		if fr.read(raw) {
			return fr.end()
		}

		var values []Value
		if values, fr.err = ValueReference(0).FromArrayBytes(raw); fr.err != nil {
			return fr.end()
		}

		for i, v := range values {
			c.Children[i] = int32(*v.(*ValueReference))
		}
	}

	c.Parents = make([]int32, instanceCount)
	if instanceCount > 0 {
		raw := make([]byte, instanceCount*4)
		if fr.read(raw) {
			return fr.end()
		}

		var values []Value
		if values, fr.err = ValueReference(0).FromArrayBytes(raw); fr.err != nil {
			return fr.end()
		}

		for i, v := range values {
			c.Parents[i] = int32(*v.(*ValueReference))
		}
	}

	return fr.end()
}

func (c *ChunkParent) WriteTo(w io.Writer) (n int64, err error) {
	fw := &formatWriter{w: w}

	if fw.writeNumber(binary.LittleEndian, c.Version) {
		return fw.end()
	}

	var instanceCount = len(c.Children)
	if fw.writeNumber(binary.LittleEndian, uint32(instanceCount)) {
		return fw.end()
	}

	if instanceCount > 0 {
		// Children
		values := make([]Value, instanceCount)
		for i, id := range c.Children {
			n := id
			values[i] = (*ValueReference)(&n)
		}

		var raw []byte
		if raw, fw.err = new(ValueReference).ArrayBytes(values); fw.err != nil {
			return fw.end()
		}

		if fw.write(raw) {
			return fw.end()
		}

		// Parents
		if len(c.Parents) != instanceCount {
			fw.err = ErrChunkParentArray
			return fw.end()
		}

		for i, id := range c.Parents {
			n := id
			values[i] = (*ValueReference)(&n)
		}

		if raw, fw.err = new(ValueReference).ArrayBytes(values); fw.err != nil {
			return fw.end()
		}

		if fw.write(raw) {
			return fw.end()
		}
	}

	return fw.end()
}

////////////////////////////////////////////////////////////////

// ChunkProperty is a Chunk that contains information about the properties of
// a group of instances.
type ChunkProperty struct {
	// Whether the chunk is compressed.
	IsCompressed bool

	// TypeID is the ID of an instance group contained in a ChunkInstance.
	TypeID int32

	// PropertyName is the name of a valid property in each instance of the
	// corresponding instance group.
	PropertyName string

	// DataType is a number indicating the type of the property.
	DataType Type

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
	return c.IsCompressed
}

func (c *ChunkProperty) SetCompressed(b bool) {
	c.IsCompressed = b
}

func (c *ChunkProperty) ReadFrom(r io.Reader) (n int64, err error) {
	fr := &formatReader{r: r}

	if fr.readNumber(binary.LittleEndian, &c.TypeID) {
		return fr.end()
	}

	if fr.readString(&c.PropertyName) {
		return fr.end()
	}

	if fr.readNumber(binary.LittleEndian, (*uint8)(&c.DataType)) {
		return fr.end()
	}

	rawBytes, failed := fr.readall()
	if failed {
		return fr.end()
	}

	newValue, ok := valueGenerators[c.DataType]
	if !ok {
		fr.err = &ErrInvalidType{Chunk: c, Bytes: rawBytes}
		return fr.end()
	}

	c.Properties, fr.err = newValue().FromArrayBytes(rawBytes)
	if fr.err != nil {
		errBytes := make([]byte, len(rawBytes))
		copy(errBytes, rawBytes)
		fr.err = ErrValue{Type: c.DataType, Bytes: errBytes, Err: fr.err}
		return fr.end()
	}

	return fr.end()
}

func (c *ChunkProperty) WriteTo(w io.Writer) (n int64, err error) {
	fw := &formatWriter{w: w}

	if fw.writeNumber(binary.LittleEndian, c.TypeID) {
		return fw.end()
	}

	if fw.writeString(c.PropertyName) {
		return fw.end()
	}

	if fw.writeNumber(binary.LittleEndian, uint8(c.DataType)) {
		return fw.end()
	}

	newValue, ok := valueGenerators[c.DataType]
	if !ok {
		fw.err = &ErrInvalidType{Chunk: c}
		return fw.end()
	}

	var rawBytes []byte
	if rawBytes, fw.err = newValue().ArrayBytes(c.Properties); fw.err != nil {
		return fw.end()
	}

	fw.write(rawBytes)

	return fw.end()
}

////////////////////////////////////////////////////////////////

// ChunkMeta is a Chunk that contains file metadata.
type ChunkMeta struct {
	// Whether the chunk is compressed.
	IsCompressed bool

	Values [][2]string
}

func newChunkMeta() Chunk {
	return new(ChunkMeta)
}

func (ChunkMeta) Signature() [4]byte {
	return [4]byte{0x4D, 0x45, 0x54, 0x41} // META
}

func (c *ChunkMeta) Compressed() bool {
	return c.IsCompressed
}

func (c *ChunkMeta) SetCompressed(b bool) {
	c.IsCompressed = b
}

type errRawBytes struct {
	Chunk Chunk
	Bytes []byte
}

func (err errRawBytes) Error() string {
	return "RAW BYTES"
}

func (c *ChunkMeta) ReadFrom(r io.Reader) (n int64, err error) {
	fr := &formatReader{r: r}

	var size uint32
	if fr.readNumber(binary.LittleEndian, &size) {
		return fr.end()
	}
	c.Values = make([][2]string, int(size))

	for i := range c.Values {
		if fr.readString(&c.Values[i][0]) {
			return fr.end()
		}
		if fr.readString(&c.Values[i][1]) {
			return fr.end()
		}
	}

	return fr.end()
}

func (c *ChunkMeta) WriteTo(w io.Writer) (n int64, err error) {
	fw := &formatWriter{w: w}

	if fw.writeNumber(binary.LittleEndian, uint32(len(c.Values))) {
		return fw.end()
	}

	for _, pair := range c.Values {
		if fw.writeString(pair[0]) {
			return fw.end()
		}
		if fw.writeString(pair[1]) {
			return fw.end()
		}
	}

	return fw.end()
}

////////////////////////////////////////////////////////////////
