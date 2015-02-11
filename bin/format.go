// Package bin implements Roblox's binary file format.
package bin

import (
	"errors"
	"github.com/robloxapi/rbxdump"
	"github.com/robloxapi/rbxfile"
	"io"
)

// Decoder decodes a FormatModel to a generic rbxfile.Root structure.
// Optionally, a rbxdump.API can be given to yield a more correct decoding.
// Without it, the decoder should attempt to decode as best as it can with no
// guarantees.
type Decoder interface {
	Decode(model *FormatModel, api *rbxdump.API) (root *rbxfile.Root, err error)
}

// Encoder encodes a rbxfile.Root structure to a FormatModel. Optionally, a
// rbxdump.API can be given to yield a more correct encoding. Without it, the
// encoder should attempt to encode as best as it can with no guarantees.
type Encoder interface {
	Encode(root *rbxfile.Root, api *rbxdump.API) (model *FormatModel, err error)
}

// Serializer implements functions that decode and encode directly between
// byte streams and rbxfile Root structures.
type Serializer struct {
	Decoder Decoder
	Encoder Encoder
}

// NewSerializer returns a new Serializer with a specified decoder and
// encoder. If either value is nil, the default RobloxCodec will be used in
// its place.
func NewSerializer(d Decoder, e Encoder) Serializer {
	s := Serializer{
		Decoder: d,
		Encoder: e,
	}

	if d == nil || e == nil {
		var codec RobloxCodec

		if d == nil {
			s.Decoder = codec
		}
		if e == nil {
			s.Encoder = codec
		}
	}

	return s
}

// Deserialize decodes data from r into a Root structure using the specified
// decoder. An optional API can be given to ensure more correct data.
func (s Serializer) Deserialize(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
	if s.Decoder == nil {
		return nil, errors.New("a decoder has not been not specified")
	}

	model := NewFormatModel()

	if _, err = model.ReadFrom(r); err != nil {
		return nil, errors.New("error parsing format: " + err.Error())
	}

	root, err = s.Decoder.Decode(model, api)
	if err != nil {
		return nil, errors.New("error decoding data: " + err.Error())
	}

	return root, nil
}

// Serialize encodes data from a Root structure to w using the specified
// encoder. An optional API can be given to ensure more correct data.
func (s Serializer) Serialize(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
	if s.Encoder == nil {
		return errors.New("an encoder has not been not specified")
	}

	model, err := s.Encoder.Encode(root, api)
	if err != nil {
		return errors.New("error encoding data: " + err.Error())
	}

	if _, err = model.WriteTo(w); err != nil {
		return errors.New("error encoding format: " + err.Error())
	}

	return nil
}

// DefaultSerializer is the serializer used by Deserialize and Serialize.
var DefaultSerializer = NewSerializer(nil, nil)

// Deserialize decodes data from r into a Root structure using the specified
// decoder. An optional API can be given to ensure more correct data.
func Deserialize(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
	return DefaultSerializer.Deserialize(r, api)
}

// Serialize encodes data from a Root structure to w using the specified
// encoder. An optional API can be given to ensure more correct data.
func Serialize(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
	return DefaultSerializer.Serialize(w, api, root)
}

// Format implements rbxfile.Format so that this package can be registered
// when it is imported. It uses the default serializer to decode and encode.
type Format struct{}

func (Format) Name() string {
	return "bin"
}

func (Format) Magic() string {
	return BinaryHeader
}

func (Format) Decode(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
	return Deserialize(r, api)
}

func (Format) Encode(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
	return Serialize(w, api, root)
}

func init() {
	rbxfile.RegisterFormat(Format{})
}
