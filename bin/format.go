// Package bin implements Roblox's binary file format.
package bin

import (
	"errors"
	"github.com/robloxapi/rbxdump"
	"github.com/robloxapi/rbxfile"
	"io"
)

// FormatDecoder decodes a FormatModel to a generic rbxfile.Root structure.
// Optionally, a rbxdump.API can be given to yield a more correct decoding.
// Without it, the decoder should attempt to decode as best as it can with no
// guarantees.
type FormatDecoder interface {
	Decode(model *FormatModel, api *rbxdump.API) (root *rbxfile.Root, err error)
}

// FormatEncoder encodes a rbxfile.Root structure to a FormatModel.
// Optionally, a rbxdump.API can be given to yield a more correct encoding.
// Without it, the encoder should attempt to encode as best as it can with no
// guarantees.
type FormatEncoder interface {
	Encode(root *rbxfile.Root, api *rbxdump.API) (model *FormatModel, err error)
}

// RobloxCodec implements FormatDecoder and FormatEncoder to emulate Roblox's
// internal codec as closely as possible.
type RobloxCodec struct{}

func (RobloxCodec) Decode(model *FormatModel) (root *rbxfile.Root, err error) {
	return nil, errors.New("not implemented")
}
func (RobloxCodec) Encode(root *rbxfile.Root) (model *FormatModel, err error) {
	return nil, errors.New("not implemented")
}

// Decode decodes data from a Reader into a Root structure using the default
// RobloxCodec.
func Decode(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
	return nil, errors.New("not implemented")
}

// Encode encodes a Root structure to a Writer using the default RobloxCodec.
func Encode(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
	return errors.New("not implemented")
}

// Format implements rbxfile.Format so that this package can be registered
// when it is imported.
type Format struct{}

func (Format) Name() string {
	return "bin"
}

func (Format) Magic() string {
	return BinaryHeader
}

func (Format) Decode(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
	return Decode(r, api)
}

func (Format) Encode(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
	return Encode(w, api, root)
}

func init() {
	rbxfile.RegisterFormat(Format{})
}
