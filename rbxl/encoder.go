package rbxl

import (
	"errors"
	"io"

	"github.com/robloxapi/rbxfile"
)

type Encoder struct {
	// Mode indicates which type of format is encoded.
	Mode Mode
}

func (e Encoder) Encode(w io.Writer, root *rbxfile.Root) (err error) {
	if w == nil {
		return errors.New("nil writer")
	}

	codec := RobloxCodec{Mode: e.Mode}
	model, err := codec.Encode(root)
	if err != nil {
		return errors.New("error encoding data: " + err.Error())
	}

	if _, err = model.WriteTo(w); err != nil {
		return errors.New("error encoding format: " + err.Error())
	}

	return nil
}
