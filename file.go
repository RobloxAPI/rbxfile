package rbxfile

import (
	"bufio"
	"errors"
	"github.com/robloxapi/rbxdump"
	"github.com/robloxapi/rbxfile/type"
	"io"
)

////////////////////////////////////////////////////////////////

// Root represents the root of an instance tree.
type Root struct {
	// Children contains instances that are children of the root.
	Children []*Instance

	// Meta contains metadata provided by the format that decoded the tree.
	Meta []string
}

// Instance represents a single Roblox instance.
type Instance struct {
	// Referent is a unique string used to refer to the instance from
	// elsewhere in the tree.
	Referent string

	// Type indicates the instance's ClassName.
	Type string

	// Properties is a map of properties of the instance. It maps the name of
	// the property to its current value.
	Properties map[string]rbxtype.Type

	// Children contains instances that are the children of the current
	// instance.
	Children []*Instance
}

////////////////////////////////////////////////////////////////

// Format encodes and decodes a single file format for a data structure.
type Format interface {
	// Name returns the name of the format.
	Name() string

	// Magic returns a magic prefix that identifies the format. The magic
	// string can contain "?" wildcards that each match any one byte.
	Magic() string

	// Decode decodes data from r into a data structure. API is an API dump
	// that can be used while decoding, and may be nil.
	Decode(r io.Reader, api *rbxdump.API) (root *Root, err error)

	// Encode encodes a data structure into w. API is a Roblox API that can be
	// used while encoding, and may be nil.
	Encode(w io.Writer, api *rbxdump.API, root *Root) (err error)
}

var formats []Format

// RegisterFormat registers a file format for use by Codec.
func RegisterFormat(format Format) {
	formats = append(formats, format)
}

////////////////////////////////////////////////////////////////

var ErrFormat = errors.New("unknown format")

// Codec encodes and decodes Roblox files using registered formats.
type Codec struct {
	// API is an API structure that can be used by formats to ensure that data
	// is encoded and decoded correctly.
	API *rbxdump.API
}

func match(magic string, b []byte) bool {
	if len(magic) != len(b) {
		return false
	}
	for i, c := range b {
		if magic[i] != c && magic[i] != '?' {
			return false
		}
	}
	return true
}

// Decode attempts to determine and decode the format of the underlying data
// stream in `r` by reading the header. Only registered formats are detected.
//
// Returns ErrFormat if the format could not be detected.
func (c *Codec) Decode(r io.Reader) (root *Root, err error) {
	var buf *bufio.Reader
	if br, ok := r.(*bufio.Reader); ok {
		buf = br
	} else {
		buf = bufio.NewReader(r)
	}

	var format Format
	for _, f := range formats {
		magic := f.Magic()
		header, err := buf.Peek(len(magic))
		if err == nil && match(magic, header) {
			format = f
		}
	}
	if format == nil {
		return nil, ErrFormat
	}

	return format.Decode(buf, c.API)
}

// Encode attempts to encode a data structure to a given format. The fmt
// argument should match the name given by the format's Name() method. Only
// registered formats can be encoded to.
//
// Returns ErrFormat if the given format is not registered.
func (c *Codec) Encode(w io.Writer, fmt string, root *Root) (err error) {
	var format Format
	for _, f := range formats {
		if fmt == f.Name() {
			format = f
			return
		}
	}
	if format == nil {
		return ErrFormat
	}

	return format.Encode(w, c.API, root)
}

////////////////////////////////////////////////////////////////

// DefaultCodec is the Codec used by Encode and Decode.
var DefaultCodec = &Codec{}

// RegisterAPI registers an API structure to be used by Encode and Decode.
func RegisterAPI(api *rbxdump.API) {
	DefaultCodec.API = api
}

// Decode attempts to determine and decode the format of the underlying data
// stream in `r` by reading the header. Only registered formats are detected.
//
// Returns ErrFormat if the format could not be detected.
func Decode(r io.Reader) (root *Root, err error) {
	return DefaultCodec.Decode(r)
}

// Encode attempts to encode a data structure to a given format. The fmt
// argument should match the name given by the format's Name() method. Only
// registered formats can be encoded to.
//
// Returns ErrFormat if the given format is not registered.
func Encode(w io.Writer, fmt string, root *Root) (err error) {
	return DefaultCodec.Encode(w, fmt, root)
}
