package rbxl

import (
	"encoding/binary"
	"fmt"
	"io"
	"unicode"

	"github.com/anaminus/parse"
	"github.com/bkaradzic/go-lz4"
)

////////////////////////////////////////////////////////////////

// robloxSig is the signature a Roblox file (binary or XML).
const robloxSig = "<roblox"

// binaryMarker indicates the start of a binary file, rather than an XML file.
const binaryMarker = "!"

// binaryHeader is the header magic of a binary file.
const binaryHeader = "\x89\xff\r\n\x1a\n"

////////////////////////////////////////////////////////////////

func readString(f *parse.BinaryReader, data *string) (failed bool) {
	if f.Err() != nil {
		return true
	}

	var length uint32
	if f.Number(&length) {
		return true
	}

	s := make([]byte, length)
	if f.Bytes(s) {
		return true
	}

	*data = string(s)

	return false
}

func writeString(f *parse.BinaryWriter, data string) (failed bool) {
	if f.Err() != nil {
		return true
	}

	if f.Number(uint32(len(data))) {
		return true
	}

	return f.Bytes([]byte(data))
}

////////////////////////////////////////////////////////////////

// validChunk returns whether a chunk signature is valid for a format version.
func validChunk(s sig) bool {
	switch s {
	case sigMETA,
		sigSSTR,
		sigINST,
		sigPROP,
		sigPRNT,
		sigEND:
		return true
	default:
		return false
	}
}

// formatModel models Roblox's binary file format. Directly, it can be used to
// control exactly how a file is encoded.
type formatModel struct {
	// Version indicates the version of the format model.
	Version uint16

	// ClassCount is the number of unique classes in the model.
	ClassCount uint32

	// InstanceCount is the number of unique instances in the model.
	InstanceCount uint32

	// Chunks is a list of Chunks present in the model.
	Chunks []chunk

	// Trailing is trailing bytes that appear after the END chunk.
	Trailing []byte

	groupLookup map[int32]*chunkInstance
}

////////////////////////////////////////////////////////////////

// sig is the signature of a chunk.
type sig uint32

func (s sig) String() string {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(s))
	for i, c := range b {
		if !unicode.IsPrint(rune(c)) {
			b[i] = '.'
		}
	}
	return string(b[:])
}

// chunk is a portion of the model that contains distinct data.
type chunk interface {
	// Signature returns a signature used to identify the chunk's type.
	Signature() sig

	// Compressed returns whether the chunk was compressed when decoding, or
	// whether thed chunk should be compressed when encoding.
	Compressed() bool

	// SetCompressed sets whether the chunk should be compressed when
	// encoding.
	SetCompressed(bool)

	// WriteTo writes the data from a chunk to an uncompressed payload. The
	// payload will be compressed afterward depending on the chunk's
	// compression settings.
	WriteTo(w io.Writer) (n int64, err error)
}

type compressed bool

func (c compressed) Compressed() bool {
	return bool(c)
}

func (c *compressed) SetCompressed(v bool) {
	*c = compressed(v)
}

// Represents a raw chunk, which contains compression data and payload.
type rawChunk struct {
	signature uint32
	compressed
	payload []byte
}

func (c rawChunk) Signature() sig {
	return sig(c.signature)
}

// Reads out a raw chunk from a stream, decompressing the chunk if necessary.
func (c *rawChunk) Decode(fr *parse.BinaryReader) bool {
	if fr.Number(&c.signature) {
		return true
	}

	var compressedLength uint32
	if fr.Number(&compressedLength) {
		return true
	}

	var decompressedLength uint32
	if fr.Number(&decompressedLength) {
		return true
	}

	var reserved uint32
	if fr.Number(&reserved) {
		return true
	}

	c.payload = make([]byte, decompressedLength)
	// If compressed length is 0, then the data is not compressed.
	if compressedLength == 0 {
		c.compressed = false
		if fr.Bytes(c.payload) {
			return true
		}
	} else {
		c.compressed = true

		// Prepare compressed data for reading by lz4, which requires the
		// uncompressed length before the compressed data.
		compressedData := make([]byte, compressedLength+4)
		binary.LittleEndian.PutUint32(compressedData, decompressedLength)

		if fr.Bytes(compressedData[4:]) {
			return true
		}

		// ROBLOX ERROR: "Malformed data ([true decompressed length] != [given
		// decompressed length])". lz4 already does some kind of size
		// validation, though the error message isn't the same.

		if _, err := lz4.Decode(c.payload, compressedData); err != nil {
			fr.Add(0, fmt.Errorf("lz4: %s", err.Error()))
			return true
		}
	}

	return false
}

// Writes a raw chunk payload to a stream, compressing if necessary.
func (c *rawChunk) WriteTo(fw *parse.BinaryWriter) bool {
	if fw.Number(c.signature) {
		return true
	}

	if c.compressed {
		var compressedData []byte
		compressedData, err := lz4.Encode(compressedData, c.payload)
		if fw.Add(0, err) {
			return true
		}

		// lz4 sanity check
		if binary.LittleEndian.Uint32(compressedData[:4]) != uint32(len(c.payload)) {
			panic("lz4 uncompressed length does not match payload length")
		}

		// Compressed length; lz4 prepends the length of the uncompressed
		// payload, so it must be excluded.
		compressedPayload := compressedData[4:]

		if fw.Number(uint32(len(compressedPayload))) {
			return true
		}

		// Decompressed length
		if fw.Number(uint32(len(c.payload))) {
			return true
		}

		// Reserved
		if fw.Number(uint32(0)) {
			return true
		}

		if fw.Bytes(compressedPayload) {
			return true
		}
	} else {
		// If the data is not compressed, then the compressed length is 0
		if fw.Number(uint32(0)) {
			return true
		}

		// Decompressed length
		if fw.Number(uint32(len(c.payload))) {
			return true
		}

		// Reserved
		if fw.Number(uint32(0)) {
			return true
		}

		if fw.Bytes(c.payload) {
			return true
		}
	}

	return false
}

////////////////////////////////////////////////////////////////

// chunkUnknown is a Chunk that is not known by the format.
type chunkUnknown struct {
	rawChunk
}

func (c *chunkUnknown) Decode(r io.Reader) (n int64, err error) {
	fr := parse.NewBinaryReader(r)

	c.payload, _ = fr.All()

	return fr.End()
}

func (c *chunkUnknown) WriteTo(w io.Writer) (n int64, err error) {
	fw := parse.NewBinaryWriter(w)

	fw.Bytes(c.payload)

	return fw.End()
}

////////////////////////////////////////////////////////////////

// chunkErrored is a chunk that has errored.
type chunkErrored struct {
	// The state of the chunk as the error occurred.
	chunk

	// Offset is the number of parsed before the error occurred.
	Offset int64

	// The error that occurred.
	Cause error

	// The raw bytes of the chunk.
	Bytes []byte
}

func (c *chunkErrored) Decode(r io.Reader) (n int64, err error) {
	fr := parse.NewBinaryReader(r)

	c.Bytes, _ = fr.All()

	return fr.End()
}

func (c *chunkErrored) WriteTo(w io.Writer) (n int64, err error) {
	fw := parse.NewBinaryWriter(w)

	fw.Bytes(c.Bytes)

	return fw.End()
}

////////////////////////////////////////////////////////////////

const sigINST = 0x54_53_4E_49 // TSNI

// chunkInstance is a Chunk that contains information about the instances in
// the file. Instances of the same ClassName are grouped together into this
// kind of chunk, which are called "instance groups".
type chunkInstance struct {
	compressed

	// ClassID is a number identifying the instance group.
	ClassID int32

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

func (chunkInstance) Signature() sig {
	return sigINST
}

func (c *chunkInstance) Decode(r io.Reader) (n int64, err error) {
	fr := parse.NewBinaryReader(r)

	if fr.Number(&c.ClassID) {
		return fr.End()
	}

	if readString(fr, &c.ClassName) {
		return fr.End()
	}

	var isService uint8
	if fr.Number(&isService) {
		return fr.End()
	}
	c.IsService = isService != 0

	var groupLength uint32
	if fr.Number(&groupLength) {
		return fr.End()
	}

	c.InstanceIDs = make([]int32, groupLength)
	if groupLength > 0 {
		raw := make([]byte, groupLength*4)
		if fr.Bytes(raw) {
			return fr.End()
		}

		values, err := refArrayFromBytes(raw, int(groupLength))
		if fr.Add(0, err) {
			return fr.End()
		}

		for i, v := range values {
			c.InstanceIDs[i] = int32(v)
		}
	}

	if c.IsService {
		c.GetService = make([]byte, groupLength)
		if fr.Bytes(c.GetService) {
			return fr.End()
		}
	}

	return fr.End()
}

func (c *chunkInstance) WriteTo(w io.Writer) (n int64, err error) {
	fw := parse.NewBinaryWriter(w)

	if fw.Number(c.ClassID) {
		return fw.End()
	}

	if writeString(fw, c.ClassName) {
		return fw.End()
	}

	var isService uint8 = 0
	if c.IsService {
		isService = 1
	}
	if fw.Number(isService) {
		return fw.End()
	}

	if fw.Number(uint32(len(c.InstanceIDs))) {
		return fw.End()
	}

	if len(c.InstanceIDs) > 0 {
		if fw.Bytes(refArrayToBytes(c.InstanceIDs)) {
			return fw.End()
		}
	}

	if c.IsService {
		if fw.Bytes(c.GetService) {
			return fw.End()
		}
	}

	return fw.End()
}

////////////////////////////////////////////////////////////////

const sigEND = 0x00_44_4E_45 // \0DNE

// chunkEnd is a Chunk that signals the end of the file. It causes the decoder
// to stop reading chunks, so it should be the last chunk.
type chunkEnd struct {
	compressed

	// The raw decompressed content of the chunk. For maximum compatibility,
	// the content should be "</roblox>", and the chunk should be
	// uncompressed. The decoder will emit warnings indicating such, if this
	// is not the case.
	Content []byte
}

func (chunkEnd) Signature() sig {
	return sigEND
}

func (c *chunkEnd) Decode(r io.Reader) (n int64, err error) {
	fr := parse.NewBinaryReader(r)

	c.Content, _ = fr.All()

	return fr.End()
}

func (c *chunkEnd) WriteTo(w io.Writer) (n int64, err error) {
	fw := parse.NewBinaryWriter(w)

	fw.Bytes(c.Content)

	return fw.End()
}

////////////////////////////////////////////////////////////////

const sigPRNT = 0x54_4E_52_50 // TNRP

// chunkParent is a Chunk that contains information about the parent-child
// relationships between instances in the model.
type chunkParent struct {
	compressed

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

func (chunkParent) Signature() sig {
	return sigPRNT
}

func (c *chunkParent) Decode(r io.Reader) (n int64, err error) {
	fr := parse.NewBinaryReader(r)

	if fr.Number(&c.Version) {
		return fr.End()
	}

	var instanceCount uint32
	if fr.Number(&instanceCount) {
		return fr.End()
	}

	c.Children = make([]int32, instanceCount)
	if instanceCount > 0 {
		raw := make([]byte, instanceCount*4)
		if fr.Bytes(raw) {
			return fr.End()
		}

		values, err := refArrayFromBytes(raw, int(instanceCount))
		if fr.Add(0, err) {
			return fr.End()
		}

		for i, v := range values {
			c.Children[i] = int32(v)
		}
	}

	c.Parents = make([]int32, instanceCount)
	if instanceCount > 0 {
		raw := make([]byte, instanceCount*4)
		if fr.Bytes(raw) {
			return fr.End()
		}

		values, err := refArrayFromBytes(raw, int(instanceCount))
		if fr.Add(0, err) {
			return fr.End()
		}

		for i, v := range values {
			c.Parents[i] = int32(v)
		}
	}

	return fr.End()
}

func (c *chunkParent) WriteTo(w io.Writer) (n int64, err error) {
	fw := parse.NewBinaryWriter(w)

	if fw.Number(c.Version) {
		return fw.End()
	}

	var instanceCount = len(c.Children)
	if len(c.Parents) != instanceCount {
		fw.Add(0, errParentArray{Children: instanceCount, Parent: len(c.Parents)})
		return fw.End()
	}
	if fw.Number(uint32(instanceCount)) {
		return fw.End()
	}
	if instanceCount > 0 {
		if fw.Bytes(refArrayToBytes(c.Children)) {
			return fw.End()
		}
		if fw.Bytes(refArrayToBytes(c.Parents)) {
			return fw.End()
		}
	}

	return fw.End()
}

////////////////////////////////////////////////////////////////

const sigPROP = 0x50_4F_52_50 // PORP

// chunkProperty is a Chunk that contains information about the properties of
// a group of instances.
type chunkProperty struct {
	compressed

	// ClassID is the ID of an instance group contained in a ChunkInstance.
	ClassID int32

	// PropertyName is the name of a valid property in each instance of the
	// corresponding instance group.
	PropertyName string

	// Properties is a list of Values of the given DataType. Each value in the
	// array corresponds to the property of an instance in the specified
	// group.
	Properties array
}

func (chunkProperty) Signature() sig {
	return sigPROP
}

func (c *chunkProperty) Decode(r io.Reader, groupLookup map[int32]*chunkInstance) (n int64, err error) {
	fr := parse.NewBinaryReader(r)

	if fr.Number(&c.ClassID) {
		return fr.End()
	}
	inst, ok := groupLookup[c.ClassID]
	if !ok {
		fr.Add(0, fmt.Errorf("unknown ID `%d`", c.ClassID))
		return fr.End()
	}

	if readString(fr, &c.PropertyName) {
		return fr.End()
	}

	rawBytes, failed := fr.All()
	if failed {
		return fr.End()
	}

	if len(rawBytes) == 0 {
		// No value data.
		c.Properties = nil
		return fr.End()
	}

	if c.Properties, _, err = typeArrayFromBytes(rawBytes, len(inst.InstanceIDs)); err != nil {
		if c.Properties != nil {
			fr.Add(0, ValueError{Type: byte(c.Properties.Type()), Cause: err})
		} else {
			fr.Add(0, err)
		}
		return fr.End()
	}

	return fr.End()
}

func (c *chunkProperty) WriteTo(w io.Writer) (n int64, err error) {
	fw := parse.NewBinaryWriter(w)

	if fw.Number(c.ClassID) {
		return fw.End()
	}

	if writeString(fw, c.PropertyName) {
		return fw.End()
	}

	if c.Properties == nil {
		// No value data.
		return fw.End()
	}

	rawBytes := make([]byte, 0, zb+c.Properties.BytesLen())
	rawBytes, err = typeArrayToBytes(rawBytes, c.Properties)
	if err != nil {
		fw.Add(0, err)
		return fw.End()
	}
	fw.Bytes(rawBytes)

	return fw.End()
}

////////////////////////////////////////////////////////////////

const sigMETA = 0x41_54_45_4D // ATEM

// chunkMeta is a Chunk that contains file metadata.
type chunkMeta struct {
	compressed

	Values [][2]string
}

func (chunkMeta) Signature() sig {
	return sigMETA
}

func (c *chunkMeta) Decode(r io.Reader) (n int64, err error) {
	fr := parse.NewBinaryReader(r)

	var size uint32
	if fr.Number(&size) {
		return fr.End()
	}
	c.Values = make([][2]string, int(size))

	for i := range c.Values {
		if readString(fr, &c.Values[i][0]) {
			return fr.End()
		}
		if readString(fr, &c.Values[i][1]) {
			return fr.End()
		}
	}

	return fr.End()
}

func (c *chunkMeta) WriteTo(w io.Writer) (n int64, err error) {
	fw := parse.NewBinaryWriter(w)

	if fw.Number(uint32(len(c.Values))) {
		return fw.End()
	}

	for _, pair := range c.Values {
		if writeString(fw, pair[0]) {
			return fw.End()
		}
		if writeString(fw, pair[1]) {
			return fw.End()
		}
	}

	return fw.End()
}

////////////////////////////////////////////////////////////////

const sigSSTR = 0x52_54_53_53 // RTSS

// chunkSharedStrings is a Chunk that contains shared strings.
type chunkSharedStrings struct {
	compressed

	Version uint32
	Values  []sharedString
}

type sharedString struct {
	Hash  [16]byte
	Value []byte
}

func (chunkSharedStrings) Signature() sig {
	return sigSSTR
}

func (c *chunkSharedStrings) Decode(r io.Reader) (n int64, err error) {
	fr := parse.NewBinaryReader(r)

	if fr.Number(&c.Version) {
		return fr.End()
	}
	// TODO: validate version?

	var length uint32
	if fr.Number(&length) {
		return fr.End()
	}
	c.Values = make([]sharedString, int(length))

	for i := range c.Values {
		if fr.Bytes(c.Values[i].Hash[:]) {
			fr.End()
		}
		var value string
		if readString(fr, &value) {
			return fr.End()
		}
		c.Values[i].Value = []byte(value)
		// TODO: validate hash?
	}

	return fr.End()
}

func (c *chunkSharedStrings) WriteTo(w io.Writer) (n int64, err error) {
	fw := parse.NewBinaryWriter(w)

	if fw.Number(c.Version) {
		return fw.End()
	}

	if fw.Number(uint32(len(c.Values))) {
		return fw.End()
	}

	for _, ss := range c.Values {
		if fw.Bytes(ss.Hash[:]) {
			fw.End()
		}
		if writeString(fw, string(ss.Value)) {
			fw.End()
		}
	}

	return fw.End()
}

////////////////////////////////////////////////////////////////
