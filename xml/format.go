package xml

import (
	"errors"
	"github.com/robloxapi/rbxdump"
	"github.com/robloxapi/rbxfile"
	"io"
)

// Decoder decodes a Document to a generic rbxfile.Root structure. Optionally,
// a rbxdump.API can be given to yield a more correct decoding. Without it,
// the decoder should attempt to decode as best as it can with no guarantees.
type Decoder interface {
	Decode(document *Document, api *rbxdump.API) (root *rbxfile.Root, err error)
}

// Encoder encodes a rbxfile.Root structure to a Document. Optionally, a
// rbxdump.API can be given to yield a more correct encoding. Without it, the
// encoder should attempt to encode as best as it can with no guarantees.
type Encoder interface {
	Encode(root *rbxfile.Root, api *rbxdump.API) (document *Document, err error)
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

	document := new(Document)

	if _, err = document.ReadFrom(r); err != nil {
		return nil, errors.New("error parsing document: " + err.Error())
	}

	root, err = s.Decoder.Decode(document, api)
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

	document, err := s.Encoder.Encode(root, api)
	if err != nil {
		return errors.New("error encoding data: " + err.Error())
	}

	if _, err = document.WriteTo(w); err != nil {
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
		name:       "rbxlx",
		magic:      "<roblox ",
		serializer: NewSerializer(rbxlCodec, rbxlCodec),
	})

	rbxmCodec := RobloxCodec{Mode: ModeModel}
	rbxfile.RegisterFormat(format{
		name:       "rbxmx",
		magic:      "<roblox ",
		serializer: NewSerializer(rbxmCodec, rbxmCodec),
	})
}
