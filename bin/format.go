// Package bin implements a decoder and encoder for Roblox's binary file
// format.
//
// This package registers the formats "rbxl" and "rbxm" to the rbxfile
// package.
//
// The easiest way to decode and encode files is through the functions
// DeserializePlace, SerializePlace, DeserializeModel, and SerializeModel.
// These decode and encode directly between byte streams and Root structures
// specified by the rbxfile package. For most purposes, this is all that is
// required to read and write Roblox binary files. Further documentation gives
// an overview of how the package works internally.
//
// Overview
//
// A Serializer is used to transform data from byte streams to Root structures
// and back. A serializer specifies a decoder and encoder. Both a decoder and
// encoder combined is referred to as a "codec".
//
// Codecs transform data between a generic rbxfile.Root structure, and this
// package's "format model" structure. Custom codecs can be implemented. For
// example, you might wish to decode files normally, but encode them in an
// alternative way:
//
//     serializer := NewSerializer(nil, CustomEncoder)
//
// Custom codecs can be used with a Serializer by implementing the Decoder and
// Encoder interfaces. Both do not need to be implemented. In the example
// above, passing nil as an argument causes the serializer to use the default
// "RobloxCodec", which implements both a default decoder and encoder. This
// codec attempts to emulate how Roblox decodes and encodes its files.
//
// A FormatModel is the representation of the file format itself, rather than
// the data it contains. The FormatModel is like a buffer between the byte
// stream and the Root structure. FormatModels can be encoded (and rarely,
// decoded) to and from Root structures in multiple ways, which is specified
// by codecs. However, there is only one way to encode and decode to and from
// a byte stream, which is handled by the FormatModel.
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

// Deserialize decodes data from r into a Root structure using the default
// decoder. Data is interpreted as a Roblox place file. An optional API can be
// given to ensure more correct data.
func DeserializePlace(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
	codec := RobloxCodec{Mode: ModePlace}
	return NewSerializer(codec, codec).Deserialize(r, api)
}

// Serialize encodes data from a Root structure to w using the default
// encoder. Data is interpreted as a Roblox place file. An optional API can be
// given to ensure more correct data.
func SerializePlace(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
	codec := RobloxCodec{Mode: ModePlace}
	return NewSerializer(codec, codec).Serialize(w, api, root)
}

// Deserialize decodes data from r into a Root structure using the default
// decoder. Data is interpreted as a Roblox model file. An optional API can be
// given to ensure more correct data.
func DeserializeModel(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
	codec := RobloxCodec{Mode: ModeModel}
	return NewSerializer(codec, codec).Deserialize(r, api)
}

// Serialize encodes data from a Root structure to w using the default
// encoder. Data is interpreted as a Roblox model file. An optional API can be
// given to ensure more correct data.
func SerializeModel(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
	codec := RobloxCodec{Mode: ModeModel}
	return NewSerializer(codec, codec).Serialize(w, api, root)
}

// Format implements rbxfile.Format so that this package can be registered
// when it is imported.
type format struct {
	name       string
	magic      string
	serializer Serializer
}

func (f format) Name() string {
	return f.name
}

func (f format) Magic() string {
	return f.magic
}

func (f format) Decode(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
	return f.serializer.Deserialize(r, api)
}

func (f format) Encode(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
	return f.serializer.Serialize(w, api, root)
}

func init() {
	// rbxl will always be chosen over rbxm when decoding, but rbxm can still
	// be encoded.
	rbxlCodec := RobloxCodec{Mode: ModePlace}
	rbxfile.RegisterFormat(format{
		name:       "rbxl",
		magic:      BinaryHeader,
		serializer: NewSerializer(rbxlCodec, rbxlCodec),
	})

	rbxmCodec := RobloxCodec{Mode: ModeModel}
	rbxfile.RegisterFormat(format{
		name:       "rbxm",
		magic:      BinaryHeader,
		serializer: NewSerializer(rbxmCodec, rbxmCodec),
	})
}
