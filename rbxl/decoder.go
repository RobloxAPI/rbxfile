package rbxl

import (
	"bytes"
	"io"

	"github.com/anaminus/parse"
	"github.com/robloxapi/rbxfile"
	"github.com/robloxapi/rbxfile/errors"
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
func (d Decoder) Decode(r io.Reader) (root *rbxfile.Root, warn, err error) {
	if r == nil {
		return nil, nil, errors.New("nil reader")
	}

	f, buf, w, err := d.decode(r)
	warn = errors.Union(warn, w)
	if err != nil {
		return nil, warn, err
	}
	if buf != nil {
		root, err = rbxlx.NewSerializer(rbxlx.RobloxCodec{}, nil).Deserialize(buf)
		if err != nil {
			return nil, warn, XMLError{Cause: err}
		}
		return root, warn, nil
	}

	// Run codec.
	codec := robloxCodec{Mode: d.Mode}
	root, w, err = codec.Decode(f)
	warn = errors.Union(warn, w)
	if err != nil {
		return nil, warn, err
	}
	return root, warn, nil
}

// Decompress reencodes the compressed chunks of the binary format as
// uncompressed. The format is decoded from r, then encoded to w.
//
// Returns ErrXML if the the data is in the legacy XML format.
func (d Decoder) Decompress(w io.Writer, r io.Reader) (warn, err error) {
	if r == nil {
		return nil, errors.New("nil reader")
	}

	f, buf, ws, err := d.decode(r)
	warn = errors.Union(warn, ws)
	if err != nil {
		return warn, err
	}
	if buf != nil {
		return warn, ErrXML
	}

	ws, err = Encoder{Mode: d.Mode, Uncompressed: true}.encode(w, f)
	warn = errors.Union(warn, ws)
	return warn, err
}

func decodeError(r *parse.BinaryReader, err error) error {
	r.Add(0, err)
	err = r.Err()
	if err != nil {
		return DataError{Offset: r.N(), Cause: err}
	}
	return nil
}

// decode parses the format. If the XML format is detected, then decode returns
// a non-nil Reader with the original content, ready to be parsed by an XML
// format decoder.
func (d Decoder) decode(r io.Reader) (f *formatModel, o io.Reader, warn, err error) {
	f = &formatModel{}
	fr := parse.NewBinaryReader(r)

	// Check signature.
	signature := make([]byte, len(robloxSig+binaryMarker))
	if fr.Bytes(signature) {
		return f, nil, nil, decodeError(fr, nil)
	}
	if !bytes.Equal(signature[:len(robloxSig)], []byte(robloxSig)) {
		return f, nil, nil, decodeError(fr, errInvalidSig)
	}

	// Check for legacy XML.
	if !bytes.Equal(signature[len(robloxSig):], []byte(binaryMarker)) {
		if d.NoXML {
			return nil, nil, nil, decodeError(fr, errInvalidSig)
		} else {
			// Reconstruct original reader.
			return nil, io.MultiReader(bytes.NewReader(signature), r), nil, nil
		}
	}

	// Check header magic.
	header := make([]byte, len(binaryHeader))
	if fr.Bytes(header) {
		return nil, nil, nil, decodeError(fr, nil)
	}
	if !bytes.Equal(header, []byte(binaryHeader)) {
		return nil, nil, nil, decodeError(fr, errors.New("the file header is corrupted"))
	}

	// Check version.
	if fr.Number(&f.Version) {
		return nil, nil, nil, decodeError(fr, nil)
	}
	if f.Version != 0 {
		return nil, nil, nil, decodeError(fr, errUnrecognizedVersion(f.Version))
	}

	if fr.Number(&f.ClassCount) {
		return nil, nil, nil, decodeError(fr, nil)
	}
	f.groupLookup = make(map[int32]*chunkInstance, f.ClassCount)

	if fr.Number(&f.InstanceCount) {
		return nil, nil, nil, decodeError(fr, nil)
	}

	var reserved [8]byte
	if fr.Bytes(reserved[:]) {
		return nil, nil, nil, decodeError(fr, nil)
	}
	var warns errors.Errors
	if reserved != [8]byte{} {
		warns = append(warns, errReserve{Offset: fr.N() - int64(len(reserved)), Bytes: reserved[:]})
	}

	for i := 0; ; i++ {
		rawChunk := new(rawChunk)
		if rawChunk.Decode(fr) {
			return nil, nil, warns.Return(), decodeError(fr, nil)
		}

		var n int64
		var err error
		var chunk chunk
		payload := bytes.NewReader(rawChunk.payload)
		switch rawChunk.signature {
		case sigMETA:
			ch := chunkMeta{}
			n, err = ch.Decode(payload)
			chunk = &ch
		case sigSSTR:
			ch := chunkSharedStrings{}
			n, err = ch.Decode(payload)
			chunk = &ch
		case sigINST:
			ch := chunkInstance{}
			n, err = ch.Decode(payload)
			chunk = &ch
			if err == nil {
				f.groupLookup[ch.ClassID] = &ch
			}
		case sigPROP:
			ch := chunkProperty{}
			n, err = ch.Decode(payload, f.groupLookup)
			chunk = &ch
		case sigPRNT:
			ch := chunkParent{}
			n, err = ch.Decode(payload)
			chunk = &ch
		case sigEND:
			ch := chunkEnd{}
			n, err = ch.Decode(payload)
			chunk = &ch
		default:
			chunk = &chunkUnknown{rawChunk: *rawChunk}
			warns = append(warns, ChunkError{Index: i, Sig: sig(rawChunk.signature), Cause: errUnknownChunkSig})
		}

		chunk.SetCompressed(bool(rawChunk.compressed))

		if err != nil {
			warns = append(warns, ChunkError{Index: i, Sig: sig(rawChunk.signature), Cause: err})
			f.Chunks = append(f.Chunks, &chunkErrored{
				chunk:  chunk,
				Offset: n,
				Cause:  err,
				Bytes:  rawChunk.payload,
			})
			continue
		}

		f.Chunks = append(f.Chunks, chunk)

		if chunk, ok := chunk.(*chunkEnd); ok {
			if chunk.Compressed() {
				warns = append(warns, errEndChunkCompressed)
			}
			if !bytes.Equal(chunk.Content, []byte("</roblox>")) {
				warns = append(warns, errEndChunkContent)
			}
			break
		}
	}

	if err = decodeError(fr, nil); err != nil {
		return nil, nil, warns.Return(), err
	}
	return f, nil, warns.Return(), nil
}
