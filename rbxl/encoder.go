package rbxl

import (
	"bytes"
	"io"

	"github.com/anaminus/parse"
	"github.com/robloxapi/rbxfile"
	"github.com/robloxapi/rbxfile/errors"
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
func (e Encoder) Encode(w io.Writer, root *rbxfile.Root) (warn, err error) {
	if w == nil {
		return nil, errors.New("nil writer")
	}

	codec := robloxCodec{Mode: e.Mode}
	f, ws, err := codec.Encode(root)
	warn = errors.Union(warn, ws)
	if err != nil {
		return warn, CodecError{Cause: err}
	}

	return e.encode(w, f)
}

func encodeError(w *parse.BinaryWriter, err error) error {
	w.Add(0, err)
	err = w.Err()
	if errs, ok := err.(errors.Errors); ok && len(errs) == 0 {
		err = nil
	}
	if err != nil {
		return DataError{Offset: w.N(), Cause: err}
	}
	return nil
}

func (e Encoder) encode(w io.Writer, f *formatModel) (warn, err error) {
	var warns errors.Errors

	fw := parse.NewBinaryWriter(w)

	if fw.Bytes([]byte(robloxSig + binaryMarker + binaryHeader)) {
		return warns.Return(), encodeError(fw, nil)
	}

	if fw.Number(f.Version) {
		return warns.Return(), encodeError(fw, nil)
	}

	if fw.Number(f.ClassCount) {
		return warns.Return(), encodeError(fw, nil)
	}

	if fw.Number(f.InstanceCount) {
		return warns.Return(), encodeError(fw, nil)
	}

	// reserved
	if fw.Number(uint64(0)) {
		return warns.Return(), encodeError(fw, nil)
	}

	for i, chunk := range f.Chunks {
		if !validChunk(f.Version, chunk.Signature()) {
			warns = append(warns, ChunkError{Index: i, Sig: chunk.Signature(), Cause: errUnknownChunkSig})
		}
		if endChunk, ok := chunk.(*chunkEnd); ok {
			if !e.Uncompressed && endChunk.IsCompressed {
				warns = append(warns, errEndChunkCompressed)
			}

			if !bytes.Equal(endChunk.Content, []byte("</roblox>")) {
				warns = append(warns, errEndChunkContent)
			}

			if i != len(f.Chunks)-1 {
				warns = append(warns, errEndChunkNotLast)
			}
		}

		rawChunk := new(rawChunk)
		rawChunk.signature = chunk.Signature()
		if !e.Uncompressed {
			rawChunk.compressed = chunk.Compressed()
		}

		buf := new(bytes.Buffer)
		if fw.Add(chunk.WriteTo(buf)) {
			return warns.Return(), encodeError(fw, nil)
		}

		rawChunk.payload = buf.Bytes()

		if rawChunk.WriteTo(fw) {
			return warns.Return(), encodeError(fw, nil)
		}
	}

	return warns.Return(), encodeError(fw, nil)
}
