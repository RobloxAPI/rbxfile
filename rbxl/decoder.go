package rbxl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"unicode"

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

// Dump writes to w a readable representation of the binary format decoded from
// r.
//
// Returns ErrXML if the the data is in the legacy XML format.
func (d Decoder) Dump(w io.Writer, r io.Reader) (warn, err error) {
	if r == nil {
		return nil, errors.New("nil reader")
	}
	if w == nil {
		return nil, errors.New("nil writer")
	}

	f, buf, ws, err := d.decode(r)
	warn = errors.Union(warn, ws)
	if err != nil {
		return warn, err
	}
	if buf != nil {
		return warn, ErrXML
	}

	classes := map[int32]string{}

	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "Version: %d", f.Version)
	fmt.Fprintf(bw, "\nClasses: %d", f.ClassCount)
	fmt.Fprintf(bw, "\nInstances: %d", f.InstanceCount)
	fmt.Fprint(bw, "\nChunks: {")
	for i, chunk := range f.Chunks {
		dumpChunk(bw, 1, i, chunk, classes)
	}
	fmt.Fprint(bw, "\n}")

	bw.Flush()
	return warn, nil
}

func dumpChunk(w *bufio.Writer, indent, i int, chunk chunk, classes map[int32]string) {
	dumpNewline(w, indent)
	if i >= 0 {
		fmt.Fprintf(w, "#%d: ", i)
	}
	dumpSig(w, chunk.Signature())
	if chunk.Compressed() {
		w.WriteString(" (compressed) {")
	} else {
		w.WriteString(" (uncompressed) {")
	}
	switch chunk := chunk.(type) {
	case *chunkMeta:
		dumpNewline(w, indent+1)
		fmt.Fprintf(w, "Count: %d", len(chunk.Values))
		for _, p := range chunk.Values {
			dumpNewline(w, indent+1)
			w.WriteByte('{')
			dumpNewline(w, indent+2)
			w.WriteString("Key: ")
			dumpString(w, indent+2, p[0])
			dumpNewline(w, indent+2)
			w.WriteString("Value: ")
			dumpString(w, indent+2, p[1])
			dumpNewline(w, indent+1)
			w.WriteByte('}')
		}
	case *chunkSharedStrings:
		dumpNewline(w, indent+1)
		fmt.Fprintf(w, "Version: %d", chunk.Version)
		dumpNewline(w, indent+1)
		fmt.Fprintf(w, "Values: (count:%d) {", len(chunk.Values))
		for _, s := range chunk.Values {
			dumpNewline(w, indent+2)
			w.WriteByte('{')
			dumpNewline(w, indent+3)
			w.WriteString("Hash: ")
			dumpBytes(w, indent+3, s.Hash[:])
			dumpNewline(w, indent+3)
			w.WriteString("Value: ")
			dumpBytes(w, indent+3, s.Value)
			dumpNewline(w, indent+2)
			w.WriteByte('}')
		}
		dumpNewline(w, indent+1)
		w.WriteByte('}')
	case *chunkInstance:
		classes[chunk.ClassID] = chunk.ClassName
		dumpNewline(w, indent+1)
		fmt.Fprintf(w, "ClassID: %d", chunk.ClassID)
		dumpNewline(w, indent+1)
		w.WriteString("ClassName: ")
		dumpString(w, indent+1, chunk.ClassName)
		if chunk.IsService {
			dumpNewline(w, indent+1)
			fmt.Fprintf(w, "InstanceIDs: (count:%d) (service) {", len(chunk.InstanceIDs))
			for i, id := range chunk.InstanceIDs {
				dumpNewline(w, indent+2)
				fmt.Fprintf(w, "%d: %d (%d)", i, id, chunk.GetService[i])
			}
			dumpNewline(w, indent+1)
			w.WriteByte('}')
		} else {
			dumpNewline(w, indent+1)
			fmt.Fprintf(w, "InstanceIDs: (count:%d) {", len(chunk.InstanceIDs))
			for i, id := range chunk.InstanceIDs {
				dumpNewline(w, indent+2)
				fmt.Fprintf(w, "%d: %d", i, id)
			}
			dumpNewline(w, indent+1)
			w.WriteByte('}')
		}
	case *chunkProperty:
		dumpNewline(w, indent+1)
		fmt.Fprintf(w, "ClassID: %d", chunk.ClassID)
		if name, ok := classes[chunk.ClassID]; ok {
			w.WriteString(" (")
			dumpString(w, indent+1, name)
			w.WriteByte(')')
		}
		dumpNewline(w, indent+1)
		w.WriteString("PropertyName: ")
		dumpString(w, indent+1, chunk.PropertyName)
		if s := chunk.DataType.String(); s == "Invalid" {
			dumpNewline(w, indent+1)
			fmt.Fprintf(w, "DataType: %d (unknown)", chunk.DataType)
		} else {
			dumpNewline(w, indent+1)
			fmt.Fprintf(w, "DataType: %d (%s)", chunk.DataType, s)
		}
		dumpNewline(w, indent+1)
		fmt.Fprintf(w, "Properties: (count:%d) {", len(chunk.Properties))
		for i, v := range chunk.Properties {
			dumpNewline(w, indent+2)
			fmt.Fprintf(w, "%d: ", i)
			v.Dump(w, indent+2)
		}
		dumpNewline(w, indent+1)
		w.WriteByte('}')
	case *chunkParent:
		dumpNewline(w, indent+1)
		fmt.Fprintf(w, "Version: %d", chunk.Version)
		dumpNewline(w, indent+1)
		w.WriteString("Values (child : parent)")
		for i, child := range chunk.Children {
			dumpNewline(w, indent+2)
			fmt.Fprintf(w, "%d : %d", child, chunk.Parents[i])
		}
	case *chunkEnd:
		dumpNewline(w, indent+1)
		w.WriteString("Content: ")
		dumpString(w, indent+1, string(chunk.Content))
	case *chunkUnknown:
		dumpNewline(w, indent+1)
		w.WriteString("<unknown chunk signature>\n\t\tBytes: ")
		dumpBytes(w, indent+1, chunk.Bytes)
	case *chunkErrored:
		dumpNewline(w, indent+1)
		w.WriteString("<errored chunk>")
		dumpNewline(w, indent+1)
		fmt.Fprintf(w, "Offset: %d", chunk.Offset)
		dumpNewline(w, indent+1)
		fmt.Fprintf(w, "Error: %s", chunk.Cause)
		dumpNewline(w, indent+1)
		w.WriteString("Chunk:")
		dumpChunk(w, indent+2, -1, chunk.chunk, classes)
		dumpNewline(w, indent+1)
		w.WriteString("Bytes: ")
		dumpBytes(w, indent+1, chunk.Bytes)
	}
	dumpNewline(w, indent)
	fmt.Fprint(w, "}")
}

func dumpNewline(w *bufio.Writer, indent int) {
	w.WriteByte('\n')
	for i := 0; i < indent; i++ {
		w.WriteByte('\t')
	}
}

func dumpSig(w *bufio.Writer, sig [4]byte) {
	for _, c := range sig {
		if unicode.IsPrint(rune(c)) {
			w.WriteByte(c)
		} else {
			w.WriteByte('.')
		}
	}
	fmt.Fprintf(w, " (% 02X)", sig)
}

func dumpString(w *bufio.Writer, indent int, s string) {
	for _, r := range s {
		if !unicode.IsGraphic(r) {
			dumpBytes(w, indent, []byte(s))
			return
		}
	}
	fmt.Fprintf(w, "(len:%d) ", len(s))
	w.WriteString(strconv.Quote(s))
}

func dumpBytes(w *bufio.Writer, indent int, b []byte) {
	fmt.Fprintf(w, "(len:%d)", len(b))
	const width = 16
	for j := 0; j < len(b); j += width {
		dumpNewline(w, indent+1)
		w.WriteString("| ")
		for i := j; i < j+width; {
			if i < len(b) {
				s := strconv.FormatUint(uint64(b[i]), 16)
				if len(s) == 1 {
					w.WriteString("0")
				}
				w.WriteString(s)
			} else if len(b) < width {
				break
			} else {
				w.WriteString("  ")
			}
			i++
			if i%8 == 0 && i < j+width {
				w.WriteString("  ")
			} else {
				w.WriteString(" ")
			}
		}
		w.WriteString("|")
		n := len(b)
		if j+width < n {
			n = j + width
		}
		for i := j; i < n; i++ {
			if 32 <= b[i] && b[i] <= 126 {
				w.WriteRune(rune(b[i]))
			} else {
				w.WriteByte('.')
			}
		}
		w.WriteByte('|')
	}
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
	sig := make([]byte, len(robloxSig+binaryMarker))
	if fr.Bytes(sig) {
		return f, nil, nil, decodeError(fr, nil)
	}
	if !bytes.Equal(sig[:len(robloxSig)], []byte(robloxSig)) {
		return f, nil, nil, decodeError(fr, errInvalidSig)
	}

	// Check for legacy XML.
	if !bytes.Equal(sig[len(robloxSig):], []byte(binaryMarker)) {
		if d.NoXML {
			return nil, nil, nil, decodeError(fr, errInvalidSig)
		} else {
			// Reconstruct original reader.
			return nil, io.MultiReader(bytes.NewReader(sig), r), nil, nil
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
		if rawChunk.ReadFrom(fr) {
			return nil, nil, warns.Return(), decodeError(fr, nil)
		}

		var n int64
		var err error
		var chunk chunk
		payload := bytes.NewReader(rawChunk.payload)
		switch string(rawChunk.signature[:]) {
		case sigMETA:
			ch := chunkMeta{}
			n, err = ch.ReadFrom(payload)
			chunk = &ch
		case sigSSTR:
			ch := chunkSharedStrings{}
			n, err = ch.ReadFrom(payload)
			chunk = &ch
		case sigINST:
			ch := chunkInstance{}
			n, err = ch.ReadFrom(payload)
			chunk = &ch
		case sigPROP:
			ch := chunkProperty{}
			n, err = ch.ReadFrom(payload)
			chunk = &ch
		case sigPRNT:
			ch := chunkParent{}
			n, err = ch.ReadFrom(payload)
			chunk = &ch
		case sigEND:
			ch := chunkEnd{}
			n, err = ch.ReadFrom(payload)
			chunk = &ch
		default:
			chunk = &chunkUnknown{Bytes: rawChunk.payload}
			warns = append(warns, ChunkError{Index: i, Sig: rawChunk.signature, Cause: errUnknownChunkSig})
		}

		chunk.SetCompressed(rawChunk.compressed)

		if err != nil {
			warns = append(warns, ChunkError{Index: i, Sig: rawChunk.signature, Cause: err})
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
