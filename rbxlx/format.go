package rbxlx

import (
	"errors"
	"github.com/robloxapi/rbxfile"
	"io"
)

// Decoder decodes a stream of bytes into a rbxfile.Root according to the rbxlx
// format.
type Decoder struct{}

// Decode reads data from r and decodes it into root.
func (Decoder) Decode(r io.Reader) (root *rbxfile.Root, err error) {
	document := new(documentRoot)
	if _, err = document.ReadFrom(r); err != nil {
		return nil, errors.New("error parsing document: " + err.Error())
	}
	codec := robloxCodec{}
	root, err = codec.Decode(document)
	if err != nil {
		return nil, errors.New("error decoding data: " + err.Error())
	}
	return root, nil
}

// Encoder encodes a rbxfile.Root into a stream of bytes according to the rbxlx
// format.
type Encoder struct{}

// Encode formats root, writing the result to w.
func (Encoder) Encode(w io.Writer, root *rbxfile.Root) (err error) {
	codec := robloxCodec{}
	document, err := codec.Encode(root)
	if err != nil {
		return errors.New("error encoding data: " + err.Error())
	}
	if _, err = document.WriteTo(w); err != nil {
		return errors.New("error encoding format: " + err.Error())
	}
	return nil
}
