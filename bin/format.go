// Package bin implements a decoder and encoder for Roblox's binary file
// format.
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
	"bufio"
	"bytes"
	"errors"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxfile"
	"github.com/robloxapi/rbxfile/xml"
	"io"
)

// Decoder decodes a FormatModel to a generic rbxfile.Root structure.
type Decoder interface {
	Decode(model *FormatModel) (root *rbxfile.Root, err error)
}

// Encoder encodes a rbxfile.Root structure to a FormatModel.
type Encoder interface {
	Encode(root *rbxfile.Root) (model *FormatModel, err error)
}

// Serializer implements functions that decode and encode directly between
// byte streams and rbxfile Root structures.
type Serializer struct {
	Decoder Decoder
	Encoder Encoder

	// DecoderXML is used to decode the legacy XML format. If DecoderXML is
	// not nil, then the serializer will attempt to detect if the stream is in
	// the XML format. If so, then it will be decoded using an xml.Serializer
	// with the given decoder.
	DecoderXML xml.Decoder
}

// NewSerializer returns a new Serializer with a specified decoder and
// encoder. If either value is nil, the default RobloxCodec will be used in
// its place. DecoderXML is set to xml.RobloxCodec.
func NewSerializer(d Decoder, e Encoder) Serializer {
	s := Serializer{
		Decoder:    d,
		Encoder:    e,
		DecoderXML: xml.RobloxCodec{},
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
func (s Serializer) Deserialize(r io.Reader) (root *rbxfile.Root, err error) {
	if s.Decoder == nil {
		return nil, errors.New("a decoder has not been not specified")
	}

	if s.DecoderXML != nil {
		var buf *bufio.Reader
		if br, ok := r.(*bufio.Reader); ok {
			buf = br
		} else {
			buf = bufio.NewReader(r)
		}

		sig, err := buf.Peek(len(RobloxSig) + len(BinaryMarker))
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(sig[:len(RobloxSig)], []byte(RobloxSig)) {
			return nil, ErrInvalidSig
		}

		if !bytes.Equal(sig[len(RobloxSig):], []byte(BinaryMarker)) {
			return xml.NewSerializer(s.DecoderXML, nil).Deserialize(buf)
		}
		r = buf
	}

	model := new(FormatModel)

	if _, err = model.ReadFrom(r); err != nil {
		return nil, errors.New("error parsing format: " + err.Error())
	}

	root, err = s.Decoder.Decode(model)
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

	model, err := s.Encoder.Encode(root)
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
func DeserializePlace(r io.Reader, api rbxapi.Root) (root *rbxfile.Root, err error) {
	codec := RobloxCodec{Mode: ModePlace, API: api}
	return Serializer{
		Encoder:    codec,
		Decoder:    codec,
		DecoderXML: xml.RobloxCodec{API: api},
	}.Deserialize(r)
}

// Serialize encodes data from a Root structure to w using the default
// encoder. Data is interpreted as a Roblox place file. An optional API can be
// given to ensure more correct data.
func SerializePlace(w io.Writer, api rbxapi.Root, root *rbxfile.Root) (err error) {
	codec := RobloxCodec{Mode: ModePlace, API: api}
	return Serializer{
		Encoder:    codec,
		Decoder:    codec,
		DecoderXML: xml.RobloxCodec{API: api},
	}.Serialize(w, root)
}

// Deserialize decodes data from r into a Root structure using the default
// decoder. Data is interpreted as a Roblox model file. An optional API can be
// given to ensure more correct data.
func DeserializeModel(r io.Reader, api rbxapi.Root) (root *rbxfile.Root, err error) {
	codec := RobloxCodec{Mode: ModeModel, API: api}
	return Serializer{
		Encoder:    codec,
		Decoder:    codec,
		DecoderXML: xml.RobloxCodec{API: api},
	}.Deserialize(r)
}

// Serialize encodes data from a Root structure to w using the default
// encoder. Data is interpreted as a Roblox model file. An optional API can be
// given to ensure more correct data.
func SerializeModel(w io.Writer, api rbxapi.Root, root *rbxfile.Root) (err error) {
	codec := RobloxCodec{Mode: ModeModel, API: api}
	return Serializer{
		Encoder:    codec,
		Decoder:    codec,
		DecoderXML: xml.RobloxCodec{API: api},
	}.Serialize(w, root)
}
