package rbxlx

import (
	"errors"
	"github.com/robloxapi/rbxfile"
	"io"
)

type Decoder struct{}

func (Decoder) Decode(r io.Reader) (root *rbxfile.Root, err error) {
	document := new(Document)
	if _, err = document.ReadFrom(r); err != nil {
		return nil, errors.New("error parsing document: " + err.Error())
	}
	codec := RobloxCodec{}
	root, err = codec.Decode(document)
	if err != nil {
		return nil, errors.New("error decoding data: " + err.Error())
	}
	return root, nil
}

type Encoder struct{}

func (Encoder) Encode(w io.Writer, root *rbxfile.Root) (err error) {
	codec := RobloxCodec{}
	document, err := codec.Encode(root)
	if err != nil {
		return errors.New("error encoding data: " + err.Error())
	}
	if _, err = document.WriteTo(w); err != nil {
		return errors.New("error encoding format: " + err.Error())
	}
	return nil
}
