// +build ignore

package rbxl

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"unicode/utf8"
)

func readFrom(f *FormatModel, b ...interface{}) (err error) {
	_, err = f.ReadFrom(bytes.NewReader(app(b...)))
	return
}

func writeTo(f *FormatModel, w *writer) (err error) {
	_, err = f.WriteTo(w)
	return
}

func app(bs ...interface{}) []byte {
	n := 0
	for _, b := range bs {
		switch b := b.(type) {
		case string:
			n += len(b)
		case []byte:
			n += len(b)
		case rune:
			n += utf8.RuneLen(b)
		case byte:
			n++
		case int:
			n++
		}
	}

	s := make([]byte, 0, n)
	for _, b := range bs {
		switch b := b.(type) {
		case string:
			s = append(s, []byte(b)...)
		case []byte:
			s = append(s, b...)
		case rune:
			s = append(s, []byte(string(b))...)
		case byte:
			s = append(s, b)
		case int:
			s = append(s, byte(b))
		}
	}

	return s
}

type writer int

func (w writer) set(n int) *writer {
	v := writer(n)
	return &v
}

func (w writer) add(n int) *writer {
	v := w + writer(n)
	return &v
}

func (w writer) copy() *writer {
	v := w
	return &v
}

func (w *writer) Write(b []byte) (n int, err error) {
	if *w <= 0 {
		return 0, io.EOF
	}

	*(*int)(w) -= len(b)

	if *w < 0 {
		return len(b) + *(*int)(w), io.ErrUnexpectedEOF
	}

	return len(b), nil
}

func pstr(s string) string {
	v := s
	return v
}

func initFormatModel() *FormatModel {
	f := new(FormatModel)
	f.Version = 0
	f.TypeCount = 3
	f.InstanceCount = 6

	names := []ValueString{
		ValueString("intValue0"),
		ValueString("intValue1"),
		ValueString("Vector3Value0"),
		ValueString("Vector3Value1"),
		ValueString("Vector3Value2"),
		ValueString("Workspace0"),
	}

	values := []ValueInt{
		ValueInt(42),
		ValueInt(-37),
	}

	f.Chunks = []Chunk{
		&ChunkInstance{
			IsCompressed: false,
			TypeID:       0,
			ClassName:    "IntValue",
			InstanceIDs:  []int32{0, 1},
			IsService:    false,
			GetService:   []byte{},
		},
		&ChunkInstance{
			IsCompressed: false,
			TypeID:       1,
			ClassName:    "Vector3Value",
			InstanceIDs:  []int32{2, 3, 4},
			IsService:    false,
			GetService:   []byte{},
		},
		&ChunkInstance{
			IsCompressed: false,
			TypeID:       2,
			ClassName:    "Workspace",
			InstanceIDs:  []int32{5},
			IsService:    true,
			GetService:   []byte{1},
		},
		&ChunkProperty{
			IsCompressed: false,
			TypeID:       0,
			PropertyName: "Name",
			DataType:     TypeString,
			Properties: []Value{
				&names[0],
				&names[1],
			},
		},
		&ChunkProperty{
			IsCompressed: false,
			TypeID:       0,
			PropertyName: "Value",
			DataType:     TypeInt,
			Properties: []Value{
				&values[0],
				&values[1],
			},
		},
		&ChunkProperty{
			IsCompressed: false,
			TypeID:       1,
			PropertyName: "Name",
			DataType:     TypeString,
			Properties: []Value{
				&names[2],
				&names[3],
				&names[4],
			},
		},
		&ChunkProperty{
			IsCompressed: false,
			TypeID:       1,
			PropertyName: "Value",
			DataType:     TypeVector3,
			Properties: []Value{
				&ValueVector3{X: 1, Y: 2, Z: 3},
				&ValueVector3{X: 4, Y: 5, Z: 6},
				&ValueVector3{X: 7, Y: 8, Z: 9},
			},
		},
		&ChunkProperty{
			IsCompressed: false,
			TypeID:       2,
			PropertyName: "Name",
			DataType:     TypeString,
			Properties: []Value{
				&names[5],
			},
		},
		&ChunkParent{
			Version:      0,
			Children:     []int32{0, 1, 2, 3, 4, 5},
			Parents:      []int32{5, 5, 0, 0, 1, -1},
			IsCompressed: false,
		},
		&ChunkEnd{
			IsCompressed: false,
			Content:      []byte("</roblox>"),
		},
	}

	return f
}

type testChunk bool

func (c testChunk) Signature() [4]byte {
	b := [4]byte{}
	copy(b[:], []byte("TEST"))
	return b
}

func (c testChunk) Compressed() bool {
	return false
}

func (c testChunk) SetCompressed(bool) {}

func (c testChunk) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, nil
}

func (c testChunk) WriteTo(w io.Writer) (n int64, err error) {
	if c {
		return 0, nil
	}
	return 0, errors.New("test write success")
}

func hasWarning(f *FormatModel, warning error) bool {
	for _, w := range f.Warnings {
		if w == warning {
			return true
		}
	}
	return false
}

func TestFormatModel_ReadFrom(t *testing.T) {
	f := new(FormatModel)
	var b []byte

	if _, err := f.ReadFrom(nil); err == nil || err.Error() != "reader is nil" {
		t.Error("expected error (nil reader), got:", err)
	}

	if err := readFrom(f); err != io.EOF {
		t.Error("expected error (no sig), got:", err)
	}
	if err := readFrom(f, RobloxSig); err != io.ErrUnexpectedEOF {
		t.Error("expected error (short sig), got:", err)
	}
	if err := readFrom(f, RobloxSig, "@"); err != ErrInvalidSig {
		t.Error("expected error (bad sig), got:", err)
	}
	b = app(b, RobloxSig, BinaryMarker)

	if err := readFrom(f, b); err != io.EOF {
		t.Error("expected error (no header), got:", err)
	}
	if err := readFrom(f, b, BinaryHeader[:1]); err != io.ErrUnexpectedEOF {
		t.Error("expected error (short header), got:", err)
	}
	if err := readFrom(f, b, make([]byte, len(BinaryHeader))); err != ErrCorruptHeader {
		t.Error("expected error (bad header), got:", err)
	}
	b = app(b, BinaryHeader)

	if err := readFrom(f, b); err != io.EOF {
		t.Error("expected error (no version), got:", err)
	}
	if err := readFrom(f, b, 0); err != io.ErrUnexpectedEOF {
		t.Error("expected error (short version), got:", err)
	}
	if err, ok := readFrom(f, b, 255, 1).(ErrUnrecognizedVersion); !ok {
		t.Error("expected error (short version), got:", err)
	} else if uint16(err) != 511 {
		t.Error("incorrect version error (expected 511), got:", uint16(err))
	}
	b = app(b, 0, 0)

	if err := readFrom(f, b); err != io.EOF {
		t.Error("expected error (no type count), got:", err)
	}
	if err := readFrom(f, b, 0, 0); err != io.ErrUnexpectedEOF {
		t.Error("expected error (short type count), got:", err)
	}
	b = app(b, 0, 0, 0, 0)

	if err := readFrom(f, b); err != io.EOF {
		t.Error("expected error (no instance count), got:", err)
	}
	if err := readFrom(f, b, 0, 0); err != io.ErrUnexpectedEOF {
		t.Error("expected error (short instance count), got:", err)
	}
	b = app(b, 0, 0, 0, 0)

	if err := readFrom(f, b); err != io.EOF {
		t.Error("expected error (no reserve), got:", err)
	}
	if err := readFrom(f, b, 0, 0, 0, 0); err != io.ErrUnexpectedEOF {
		t.Error("expected error (short reserve), got:", err)
	}
	if err := readFrom(f, b, 1, 0, 0, 0, 0, 0, 0, 0); err != io.EOF {
		t.Error("expected error (no chunks), got:", err)
	}
	if len(f.Warnings) == 0 {
		t.Error("expected warning (non-zero reserve)")
	} else if f.Warnings[0] != WarnReserveNonZero {
		t.Error("expected warning (non-zero reserve), got:", f.Warnings[0])
	}
	b = app(b, 0, 0, 0, 0, 0, 0, 0, 0)

	if err := readFrom(f, b, "TEST"); err != io.EOF {
		t.Error("expected error (bad raw chunk), got:", err)
	}

	if err := readFrom(f, b, "TEST", 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0); err != io.EOF {
		t.Error("expected error (no extra chunk), got:", err)
	}
	if len(f.Warnings) == 0 {
		t.Error("expected warning (bad chunk sig)")
	} else if _, ok := f.Warnings[0].(WarnUnknownChunk); !ok {
		t.Error("expected warning (bad chunk sig), got:", f.Warnings[0])
	}

	if err := readFrom(f, b, "END\x00", 18, 0, 0, 0, 16, 0, 0, 0, 0, 0, 0, 0, []byte{240, 1, 101, 110, 100, 32, 116, 101, 115, 116, 32, 99, 111, 110, 116, 101, 110, 116}); err != nil {
		t.Error("unexpected error:", err)
	}
	if len(f.Warnings) == 0 {
		t.Error("expected warning (compressed end chunk)")
	} else if f.Warnings[0] != WarnEndChunkCompressed {
		t.Error("expected warning (compressed end chunk), got:", f.Warnings[0])
	}
	if len(f.Chunks) == 0 {
		t.Error("expected chunk")
	} else if chunk, ok := f.Chunks[0].(*ChunkEnd); !ok {
		t.Error("expected end chunk")
	} else {
		if string(chunk.Content) != "end test content" {
			t.Error("unexpected chunk payload, got:", string(chunk.Content))
		}
		if !chunk.IsCompressed {
			t.Error("unexpected chunk chunk compression")
		}
	}

	if err, ok := readFrom(f, b, "INST", 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0).(ErrChunk); !ok {
		t.Error("expected error (empty inst chunk), got:", err)
	} else {
		if string(err.Sig[:]) != "INST" {
			t.Error("expected INST chunk error, got:", string(err.Sig[:]))
		}
		if err.Err != io.EOF {
			t.Error("expected EOF chunk error, got:", err.Err)
		}
	}
}

func TestFormatModel_WriteTo(t *testing.T) {
	w := new(writer)
	f := initFormatModel()

	if _, err := f.WriteTo(nil); err == nil || err.Error() != "writer is nil" {
		t.Error("expected error (nil writer), got:", err)
	}

	// Signature
	if err := writeTo(f, w.copy()); err == nil || err != io.EOF {
		t.Error("expected error (no sig space), got:", err)
	}
	if err := writeTo(f, w.add(len(RobloxSig))); err == nil || err != io.ErrUnexpectedEOF {
		t.Error("expected error (short sig space), got:", err)
	}
	w = w.add(len(RobloxSig + BinaryMarker + BinaryHeader))

	// Version
	if err := writeTo(f, w.copy()); err == nil || err != io.EOF {
		t.Error("expected error (no version space), got:", err)
	}
	if err := writeTo(f, w.add(1)); err == nil || err != io.ErrUnexpectedEOF {
		t.Error("expected error (short version space), got:", err)
	}
	w = w.add(2)

	// TypeCount
	if err := writeTo(f, w.copy()); err == nil || err != io.EOF {
		t.Error("expected error (no type count space), got:", err)
	}
	if err := writeTo(f, w.add(2)); err == nil || err != io.ErrUnexpectedEOF {
		t.Error("expected error (short type count space), got:", err)
	}
	w = w.add(4)

	// InstanceCount
	if err := writeTo(f, w.copy()); err == nil || err != io.EOF {
		t.Error("expected error (no instance count space), got:", err)
	}
	if err := writeTo(f, w.add(2)); err == nil || err != io.ErrUnexpectedEOF {
		t.Error("expected error (short instance count space), got:", err)
	}
	w = w.add(4)

	// Reserve
	if err := writeTo(f, w.copy()); err == nil || err != io.EOF {
		t.Error("expected error (no reserve space), got:", err)
	}
	if err := writeTo(f, w.add(4)); err == nil || err != io.ErrUnexpectedEOF {
		t.Error("expected error (short reserve space), got:", err)
	}
	w = w.add(8)

	// Chunks
	if err := writeTo(f, w.copy()); err == nil || err != io.EOF {
		t.Error("expected error (no chunk space), got:", err)
	}
	if err := writeTo(f, w.add(1)); err == nil || err != io.ErrUnexpectedEOF {
		t.Error("expected error (short chunk space), got:", err)
	}
	w = w.add(1 << 10)

	chunk, _ := f.Chunks[len(f.Chunks)-1].(*ChunkEnd)
	chunk.IsCompressed = true
	chunk.Content = []byte("test content")

	f.Chunks = append(f.Chunks, testChunk(true))

	if err := writeTo(f, w.copy()); err != nil {
		t.Error("unexpected error (chunk warnings):", err)
	} else {
		if !hasWarning(f, WarnEndChunkCompressed) {
			t.Error("expected warning (compressed end chunk)")
		}
		if !hasWarning(f, WarnEndChunkContent) {
			t.Error("expected warning (bad end chunk content)")
		}
		if !hasWarning(f, WarnEndChunkNotLast) {
			t.Error("expected warning (end chunk not last)")
		}
		for _, warning := range f.Warnings {
			if warning, ok := warning.(WarnUnknownChunk); ok {
				if string(warning[:]) != "TEST" {
					t.Error("unexpected signature (unknown chunk)", [4]byte(warning))
				}
				goto okay
			}
		}
		t.Error("expected warning (unknown chunk)")
	okay:
	}

	f.Chunks[8].(*ChunkParent).Parents = []int32{}
	if err := writeTo(f, w.copy()); err == nil || err != ErrChunkParentArray {
		t.Error("expected error (chunk write), got:", err)
	}
}
