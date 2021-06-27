package rbxl

import (
	"bytes"
	"errors"
	"io"

	"github.com/anaminus/parse"
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

	var f formatModel
	fr := parse.NewBinaryReader(r)

	// Check signature.
	sig := make([]byte, len(robloxSig+binaryMarker))
	if fr.Bytes(sig) {
		return nil, fr.Err()
	}
	if !bytes.Equal(sig[:len(robloxSig)], []byte(robloxSig)) {
		return nil, ErrInvalidSig
	}

	// Check for legacy XML.
	if !bytes.Equal(sig[len(robloxSig):], []byte(binaryMarker)) {
		if d.NoXML {
			return nil, ErrInvalidSig
		} else {
			// Reconstruct original reader.
			buf := io.MultiReader(bytes.NewReader(sig), r)
			return rbxlx.NewSerializer(rbxlx.RobloxCodec{}, nil).Deserialize(buf)
		}
	}

	// Check header magic.
	header := make([]byte, len(binaryHeader))
	if fr.Bytes(header) {
		return nil, fr.Err()
	}
	if !bytes.Equal(header, []byte(binaryHeader)) {
		return nil, ErrCorruptHeader
	}

	// Check version.
	if fr.Number(&f.Version) {
		return nil, fr.Err()
	}
	switch f.Version {
	default:
		return nil, ErrUnrecognizedVersion(f.Version)
	case 0:
		err = d.version0(fr, &f)
	}
	if err != nil {
		return nil, err
	}

	// Run codec.
	codec := RobloxCodec{Mode: d.Mode}
	root, err = codec.Decode(&f)
	if err != nil {
		return nil, errors.New("error decoding data: " + err.Error())
	}

	return root, nil
}

func (d Decoder) version0(fr *parse.BinaryReader, f *formatModel) (err error) {
	f.Warnings = f.Warnings[:0]
	f.Chunks = f.Chunks[:0]

	if fr.Number(&f.ClassCount) {
		return fr.Err()
	}

	if fr.Number(&f.InstanceCount) {
		return fr.Err()
	}

	var reserved uint64
	if fr.Number(&reserved) {
		return fr.Err()
	}
	if reserved != 0 {
		f.Warnings = append(f.Warnings, WarnReserveNonZero)
	}

loop:
	for {
		rawChunk := new(rawChunk)
		if rawChunk.ReadFrom(fr) {
			return fr.Err()
		}

		newChunk := chunkGenerators(f.Version, rawChunk.signature)
		if newChunk == nil {
			newChunk = newChunkUnknown
		}
		chunk := newChunk()
		chunk.SetCompressed(rawChunk.compressed)

		if _, err := chunk.ReadFrom(bytes.NewReader(rawChunk.payload)); err != nil {
			err = ErrChunk{Sig: rawChunk.signature, Err: err}
			if f.Strict {
				fr.Add(0, err)
				return fr.Err()
			}
			f.Warnings = append(f.Warnings, err)
			continue loop
		}

		f.Chunks = append(f.Chunks, chunk)

		switch chunk := chunk.(type) {
		case *chunkUnknown:
			f.Warnings = append(f.Warnings, chunk)
		case *chunkEnd:
			if chunk.Compressed() {
				f.Warnings = append(f.Warnings, WarnEndChunkCompressed)
			}

			if !bytes.Equal(chunk.Content, []byte("</roblox>")) {
				f.Warnings = append(f.Warnings, WarnEndChunkContent)
			}

			break loop
		}
	}

	return fr.Err()
}
