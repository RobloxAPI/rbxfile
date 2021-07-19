package rbxl

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"unicode"

	"github.com/robloxapi/rbxfile/errors"
)

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
		if chunk.Properties != nil {
			t := chunk.Properties.Type()
			length := chunk.Properties.Len()
			dumpNewline(w, indent+1)
			fmt.Fprintf(w, "Properties: (type:%d (%s)) (count:%d) {", t, t.String(), length)
			for i := 0; i < length; i++ {
				v := chunk.Properties.Get(i)
				dumpNewline(w, indent+2)
				fmt.Fprintf(w, "%d: ", i)
				v.Dump(w, indent+2)
			}
			dumpNewline(w, indent+1)
			w.WriteByte('}')
		}
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
		dumpBytes(w, indent+1, chunk.payload)
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

func dumpSig(w *bufio.Writer, sig sig) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(sig))
	for _, c := range b {
		if unicode.IsPrint(rune(c)) {
			w.WriteByte(c)
		} else {
			w.WriteByte('.')
		}
	}
	fmt.Fprintf(w, " (% 02X)", b)
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
