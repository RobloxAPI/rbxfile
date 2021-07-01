package rbxl

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	// Indicates an unexpected file signature.
	ErrInvalidSig = errors.New("invalid signature")
	// Indicates the unexpected detection of the legacy XML format.
	ErrXML = errors.New("unexpected XML format")
	// Indicates unexpected header content.
	ErrCorruptHeader = errors.New("the file header is corrupted")
	// Indicates a chunk signature not known by the codec.
	ErrUnknownChunkSig = errors.New("unknown chunk signature")
	// Indicates
	ErrChunkParentArray = errors.New("length of parent array does not match children array")
	// Indicates an unexpected content for bytes presumed to be reserved.
	ErrReserveNonZero = errors.New("reserved space in file header is non-zero")
	// Indicates that the end chunk is compressed, where it is expected to be
	// uncompressed.
	ErrEndChunkCompressed = errors.New("end chunk is compressed")
	// Indicates unexpected content within the end chunk.
	ErrEndChunkContent = errors.New("end chunk content is not `</roblox>`")
	// Indicates that there are additional chunks that follow the end chunk.
	ErrEndChunkNotLast = errors.New("end chunk is not the last chunk")
)

// ErrUnrecognizedVersion indicates a format version not recognized by the
// codec.
type ErrUnrecognizedVersion uint16

func (err ErrUnrecognizedVersion) Error() string {
	return fmt.Sprintf("unrecognized version %d", err)
}

// ErrUnknownType indicates a property data type not known by the codec.
type ErrUnknownType typeID

func (err ErrUnknownType) Error() string {
	return fmt.Sprintf("unknown data type 0x%X", byte(err))
}

// ErrValue is an error that is produced by a Value of a certain Type.
type ErrValue struct {
	Type byte

	Cause error
}

func (err ErrValue) Error() string {
	return fmt.Sprintf("type %s (0x%X): %s", typeID(err.Type).String(), err.Type, err.Cause.Error())
}

func (err ErrValue) Unwrap() error {
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
	Sig [4]byte

	Cause error
}

func (err ChunkError) Error() string {
	if err.Index < 0 {
		return fmt.Sprintf("%q chunk: %s", string(err.Sig[:]), err.Cause.Error())
	}
	return fmt.Sprintf("#%d %q chunk: %s", err.Index, string(err.Sig[:]), err.Cause.Error())
}

func (err ChunkError) Unwrap() error {
	return err.Cause
}
