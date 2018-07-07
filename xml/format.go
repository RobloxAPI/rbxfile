package xml

import (
	"errors"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxfile"
	"io"
)

// Decoder decodes a Document to a generic rbxfile.Root structure.
type Decoder interface {
	Decode(document *Document) (root *rbxfile.Root, err error)
}

// Encoder encodes a rbxfile.Root structure to a Document.
type Encoder interface {
	Encode(root *rbxfile.Root) (document *Document, err error)
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
// decoder.
func (s Serializer) Deserialize(r io.Reader) (root *rbxfile.Root, err error) {
	if s.Decoder == nil {
		return nil, errors.New("a decoder has not been not specified")
	}

	document := new(Document)

	if _, err = document.ReadFrom(r); err != nil {
		return nil, errors.New("error parsing document: " + err.Error())
	}

	root, err = s.Decoder.Decode(document)
	if err != nil {
		return nil, errors.New("error decoding data: " + err.Error())
	}

	return root, nil
}

// Serialize encodes data from a Root structure to w using the specified
// encoder.
func (s Serializer) Serialize(w io.Writer, root *rbxfile.Root) (err error) {
	if s.Encoder == nil {
		return errors.New("an encoder has not been not specified")
	}

	document, err := s.Encoder.Encode(root)
	if err != nil {
		return errors.New("error encoding data: " + err.Error())
	}

	if _, err = document.WriteTo(w); err != nil {
		return errors.New("error encoding format: " + err.Error())
	}

	return nil
}

// Deserialize decodes data from r into a Root structure using the default
// decoder. An optional API can be given to ensure more correct data.
func Deserialize(r io.Reader, api rbxapi.Root) (root *rbxfile.Root, err error) {
	codec := RobloxCodec{API: api}
	return NewSerializer(codec, codec).Deserialize(r)
}

// Serialize encodes data from a Root structure to w using the default
// encoder. An optional API can be given to ensure more correct data.
func Serialize(w io.Writer, api rbxapi.Root, root *rbxfile.Root) (err error) {
	codec := RobloxCodec{API: api}
	return NewSerializer(codec, codec).Serialize(w, root)
}
