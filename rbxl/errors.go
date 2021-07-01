package rbxl

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	// Indicates an unexpected file signature.
	errInvalidSig = errors.New("invalid signature")
	// Indicates a chunk signature not known by the codec.
	errUnknownChunkSig = errors.New("unknown chunk signature")
	// Indicates that the end chunk is compressed, where it is expected to be
	// uncompressed.
	errEndChunkCompressed = errors.New("end chunk is compressed")
	// Indicates unexpected content within the end chunk.
	errEndChunkContent = errors.New("end chunk content is not `</roblox>`")
	// Indicates that there are additional chunks that follow the end chunk.
	errEndChunkNotLast = errors.New("end chunk is not the last chunk")
)

// errUnrecognizedVersion indicates a format version not recognized by the
// codec.
type errUnrecognizedVersion uint16

func (err errUnrecognizedVersion) Error() string {
	return fmt.Sprintf("unrecognized version %d", err)
}

// errUnknownType indicates a property data type not known by the codec.
type errUnknownType typeID

func (err errUnknownType) Error() string {
	return fmt.Sprintf("unknown data type 0x%X", byte(err))
}

// errReserve indicates an unexpected value for bytes that are presumed to be
// reserved.
type errReserve struct {
	// Offset marks the location of the reserved bytes.
	Offset int64
	// Bytes is the unexpected content of the reserved bytes.
	Bytes []byte
}

func (err errReserve) Error() string {
	return fmt.Sprintf("unexpected content for reserved bytes near %d: % 02X", err.Offset, err.Bytes)
}

type errParentArray struct {
	Children int
	Parent   int
}

func (err errParentArray) Error() string {
	return fmt.Sprintf("length of parents array (%d) does not match length of children array (%d)", err.Parent, err.Children)
}

// ErrXML indicates the unexpected detection of the legacy XML format.
var ErrXML = errors.New("unexpected XML format")

// ValueError is an error that is produced by a Value of a certain Type.
type ValueError struct {
	Type byte

	Cause error
}

func (err ValueError) Error() string {
	return fmt.Sprintf("type %s (0x%X): %s", typeID(err.Type).String(), err.Type, err.Cause.Error())
}

func (err ValueError) Unwrap() error {
	return err.Cause
}

// XMLError wraps an error that occurred while parsing the legacy XML format.
type XMLError struct {
	Cause error
}

func (err XMLError) Error() string {
	if err.Cause == nil {
		return "decoding XML"
	}
	return "decoding XML: " + err.Cause.Error()
}

func (err XMLError) Unwrap() error {
	return err.Cause
}

// CodecError wraps an error that occurred while encoding or decoding a binary
// format model.
type CodecError struct {
	Cause error
}

func (err CodecError) Error() string {
	if err.Cause == nil {
		return "codec error"
	}
	return "codec error: " + err.Cause.Error()
}

func (err CodecError) Unwrap() error {
	return err.Cause
}

// DataError wraps an error that occurred while encoding or decoding byte data.
type DataError struct {
	// Offset is the byte offset where the error occurred.
	Offset int64

	Cause error
}

func (err DataError) Error() string {
	var s strings.Builder
	s.WriteString("data error")
	if err.Offset >= 0 {
		s.WriteString(" at ")
		s.Write(strconv.AppendInt(nil, err.Offset, 10))
	}
	if err.Cause != nil {
		s.WriteString(": ")
		s.WriteString(err.Cause.Error())
	}
	return s.String()
}

func (err DataError) Unwrap() error {
	return err.Cause
}

// ChunkError indicates an error that occurred within a chunk.
type ChunkError struct {
	// Index is the position of the chunk within the file.
	Index int
	// Sig is the signature of the chunk.
	Sig uint32

	Cause error
}

func (err ChunkError) Error() string {
	var sig [4]byte
	binary.LittleEndian.PutUint32(sig[:], err.Sig)
	if err.Index < 0 {
		return fmt.Sprintf("%q chunk: %s", string(sig[:]), err.Cause.Error())
	}
	return fmt.Sprintf("#%d %q chunk: %s", err.Index, string(sig[:]), err.Cause.Error())
}

func (err ChunkError) Unwrap() error {
	return err.Cause
}
