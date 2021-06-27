package rbxl

import (
	"bufio"
	"bytes"
	"errors"
	"io"

	"github.com/robloxapi/rbxfile"
	"github.com/robloxapi/rbxfile/rbxlx"
)

// Decoder decodes a stream of bytes into an rbxfile.Root.
type Decoder struct {
	// Mode indicates which type of format is decoded.
	Mode Mode

	// If NoXML is true, then the decoder will not attempt to decode the legacy
	// XML format for backward compatibility.
	NoXML bool
}

// Decode reads data from r and decodes it into root according to the rbxl
// format.
func (d Decoder) Decode(r io.Reader) (root *rbxfile.Root, err error) {
	if r == nil {
		return nil, errors.New("nil reader")
	}

	if !d.NoXML {
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
			return rbxlx.NewSerializer(rbxlx.RobloxCodec{}, nil).Deserialize(buf)
		}
		r = buf
	}

	model := new(FormatModel)
	if _, err = model.ReadFrom(r); err != nil {
		return nil, errors.New("error parsing format: " + err.Error())
	}

	codec := RobloxCodec{Mode: d.Mode}
	root, err = codec.Decode(model)
	if err != nil {
		return nil, errors.New("error decoding data: " + err.Error())
	}

	return root, nil
}
