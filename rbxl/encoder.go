package rbxl

import (
	"bytes"
	"errors"
	"io"

	"github.com/anaminus/parse"
	"github.com/robloxapi/rbxfile"
)

// Encoder encodes a rbxfile.Root into a stream of bytes.
type Encoder struct {
	// Mode indicates which type of format is encoded.
	Mode Mode

	// Uncompressed sets whether compression is forcibly disabled for all
	// chunks.
	Uncompressed bool
}

// Encode formats root according to the rbxl format, and writers it to w.
func (e Encoder) Encode(w io.Writer, root *rbxfile.Root) (err error) {
	if w == nil {
		return errors.New("nil writer")
	}

	codec := robloxCodec{Mode: e.Mode}
	f, err := codec.Encode(root)
	if err != nil {
		return errors.New("error encoding data: " + err.Error())
	}

	f.Warnings = f.Warnings[:0]

	fw := parse.NewBinaryWriter(w)

	if fw.Bytes([]byte(robloxSig + binaryMarker + binaryHeader)) {
		return fw.Err()
	}

	if fw.Number(f.Version) {
		return fw.Err()
	}

	if fw.Number(f.ClassCount) {
		return fw.Err()
	}

	if fw.Number(f.InstanceCount) {
		return fw.Err()
	}

	// reserved
	if fw.Number(uint64(0)) {
		return fw.Err()
	}

	for i, chunk := range f.Chunks {
		if !validChunk(f.Version, chunk.Signature()) {
			f.Warnings = append(f.Warnings, &chunkUnknown{
				Sig: chunk.Signature(),
			})
		}

		if endChunk, ok := chunk.(*chunkEnd); ok {
			if !e.Uncompressed && endChunk.IsCompressed {
				f.Warnings = append(f.Warnings, WarnEndChunkCompressed)
			}

			if !bytes.Equal(endChunk.Content, []byte("</roblox>")) {
				f.Warnings = append(f.Warnings, WarnEndChunkContent)
			}

			if i != len(f.Chunks)-1 {
				f.Warnings = append(f.Warnings, WarnEndChunkNotLast)
			}
		}

		rawChunk := new(rawChunk)
		rawChunk.signature = chunk.Signature()
		if !e.Uncompressed {
			rawChunk.compressed = chunk.Compressed()
		}

		buf := new(bytes.Buffer)
		if fw.Add(chunk.WriteTo(buf)) {
			return fw.Err()
		}

		rawChunk.payload = buf.Bytes()

		if rawChunk.WriteTo(fw) {
			return fw.Err()
		}
	}

	return fw.Err()
}
