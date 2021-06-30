package rbxl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode"

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

	buf, err := d.decode(fr, &f)
	if err != nil {
		return nil, err
	}
	if buf != nil {
		return rbxlx.NewSerializer(rbxlx.RobloxCodec{}, nil).Deserialize(buf)
	}

	// Run codec.
	codec := robloxCodec{Mode: d.Mode}
	root, err = codec.Decode(&f)
	if err != nil {
		return nil, errors.New("error decoding data: " + err.Error())
	}

	return root, nil
}

var ErrDumpXML = errors.New("detected XML format")

// Decompress reencodes the compressed chunks of the binary format as
// uncompressed. The format is decoded from r, then encoded to w.
//
// Returns ErrDumpXML if the the data is in the legacy XML format.
func (d Decoder) Decompress(w io.Writer, r io.Reader) (err error) {
	if r == nil {
		return errors.New("nil reader")
	}

	var f formatModel
	fr := parse.NewBinaryReader(r)

	buf, err := d.decode(fr, &f)
	if err != nil {
		return err
	}
	if buf != nil {
		return ErrDumpXML
	}

	return Encoder{Mode: d.Mode, Uncompressed: true}.encode(w, &f)
}

// Dump writes to w a readable representation of the binary format decoded from
// r.
//
// Returns ErrDumpXML if the the data is in the legacy XML format.
func (d Decoder) Dump(w io.Writer, r io.Reader) (err error) {
	if r == nil {
		return errors.New("nil reader")
	}

	var f formatModel
	fr := parse.NewBinaryReader(r)

	buf, err := d.decode(fr, &f)
	if err != nil {
		return err
	}
	if buf != nil {
		return ErrDumpXML
	}

	classes := map[int32]string{}

	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "Version: %d", f.Version)
	fmt.Fprintf(bw, "\nClasses: %d", f.ClassCount)
	fmt.Fprintf(bw, "\nInstances: %d", f.InstanceCount)
	fmt.Fprint(bw, "\nChunks: {")
	for _, chunk := range f.Chunks {
		dumpNewline(bw, 1)
		dumpSig(bw, chunk.Signature())
		if chunk.Compressed() {
			bw.WriteString(" (compressed) {")
		} else {
			bw.WriteString(" (uncompressed) {")
		}
		switch chunk := chunk.(type) {
		case *chunkMeta:
			fmt.Fprintf(bw, "\n\t\tCount: %d", len(chunk.Values))
			for _, p := range chunk.Values {
				bw.WriteString("\n\t\t{\n\t\t\tKey: ")
				dumpString(bw, 3, p[0])
				bw.WriteString("\n\t\t\tValue: ")
				dumpString(bw, 3, p[1])
				bw.WriteString("\n\t\t}")
			}
		case *chunkSharedStrings:
			fmt.Fprintf(bw, "\n\t\tVersion: %d", chunk.Version)
			fmt.Fprintf(bw, "\n\t\tValues: (count:%d) {", len(chunk.Values))
			for _, s := range chunk.Values {
				bw.WriteString("\n\t\t\t{\n\t\t\t\tHash: ")
				dumpBytes(bw, 4, s.Hash[:])
				bw.WriteString("\n\t\t\t\tValue: ")
				dumpBytes(bw, 4, s.Value)
				bw.WriteString("\n\t\t\t}")
			}
			bw.WriteString("\n\t\t}")
		case *chunkInstance:
			classes[chunk.ClassID] = chunk.ClassName
			fmt.Fprintf(bw, "\n\t\tClassID: %d", chunk.ClassID)
			bw.WriteString("\n\t\tClassName: ")
			dumpString(bw, 2, chunk.ClassName)
			if chunk.IsService {
				fmt.Fprintf(bw, "\n\t\tInstanceIDs: (count:%d) (service) {", len(chunk.InstanceIDs))
				for i, id := range chunk.InstanceIDs {
					fmt.Fprintf(bw, "\n\t\t\t%d: %d (%d)", i, id, chunk.GetService[i])
				}
				bw.WriteString("\n\t\t}")
			} else {
				fmt.Fprintf(bw, "\n\t\tInstanceIDs: (count:%d) {", len(chunk.InstanceIDs))
				for i, id := range chunk.InstanceIDs {
					fmt.Fprintf(bw, "\n\t\t\t%d: %d", i, id)
				}
				bw.WriteString("\n\t\t}")
			}
		case *chunkProperty:
			fmt.Fprintf(bw, "\n\t\tClassID: %d", chunk.ClassID)
			if name, ok := classes[chunk.ClassID]; ok {
				bw.WriteString(" (")
				dumpString(bw, 2, name)
				bw.WriteByte(')')
			}
			bw.WriteString("\n\t\tPropertyName: ")
			dumpString(bw, 2, chunk.PropertyName)
			if s := chunk.DataType.String(); s == "Invalid" {
				fmt.Fprintf(bw, "\n\t\tDataType: %d (unknown)", chunk.DataType)
			} else {
				fmt.Fprintf(bw, "\n\t\tDataType: %d (%s)", chunk.DataType, s)
			}
			fmt.Fprintf(bw, "\n\t\tProperties: (count:%d) {", len(chunk.Properties))
			for i, v := range chunk.Properties {
				fmt.Fprintf(bw, "\n\t\t\t%d: ", i)
				v.Dump(bw, 3)
			}
			bw.WriteString("\n\t\t}")
		case *chunkParent:
			fmt.Fprintf(bw, "\n\t\tVersion: %d", chunk.Version)
			bw.WriteString("\n\t\tValues (child : parent)")
			for i, child := range chunk.Children {
				fmt.Fprintf(bw, "\n\t\t\t%d : %d", child, chunk.Parents[i])
			}
		case *chunkEnd:
			bw.WriteString("\n\t\tContent: ")
			dumpString(bw, 2, string(chunk.Content))
		case *chunkUnknown:
			bw.WriteString("\n\t\tUnknown Chunk Signature\n\t\tBytes: ")
			dumpBytes(bw, 2, chunk.Bytes)
		}
		fmt.Fprint(bw, "\n\t}")
	}
	fmt.Fprint(bw, "\n}")

	bw.Flush()
	return nil
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

// decode parses the format. If the XML format is detected, then decode returns
// a non-nil Reader with the original content, ready to be parsed by an XML
// format decoder.
func (d Decoder) decode(fr *parse.BinaryReader, f *formatModel) (r io.Reader, err error) {
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
			r = io.MultiReader(bytes.NewReader(sig), r)
			return r, nil
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
		err = d.version0(fr, f)
	}
	return nil, err
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
