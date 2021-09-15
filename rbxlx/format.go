package rbxlx

import (
	"fmt"
	"io"

	"github.com/robloxapi/rbxfile"
)

// Decoder decodes a stream of bytes into a rbxfile.Root according to the rbxlx
// format.
type Decoder struct {
	// DiscardInvalidProperties determines how invalid properties are decoded.
	// If true, when the parser successfully decodes a property, but fails to
	// decode its value or a component, then the entire property is discarded.
	// If false, then as much information as possible is retained; any value or
	// component that fails will be emitted as the zero value for the type.
	DiscardInvalidProperties bool
}

// Decode reads data from r and decodes it into root.
func (d Decoder) Decode(r io.Reader) (root *rbxfile.Root, warn, err error) {
	document := new(documentRoot)
	if _, err = document.ReadFrom(r); err != nil {
		return nil, document.Warnings.Return(), fmt.Errorf("error parsing document: %w", err)
	}
	codec := robloxCodec{
		DiscardInvalidProperties: d.DiscardInvalidProperties,
	}
	root, err = codec.Decode(document)
	if err != nil {
		return nil, document.Warnings.Return(), fmt.Errorf("error decoding data: %w", err)
	}
	return root, document.Warnings.Return(), nil
}

// Encoder encodes a rbxfile.Root into a stream of bytes according to the rbxlx
// format.
type Encoder struct {
	// ExcludeReferent determines whether the "referent" attribute should be
	// excluded from Item tags when encoding.
	ExcludeReferent bool

	// ExcludeExternal determines whether standard <External> tags should be
	// excluded from the root tag when encoding.
	ExcludeExternal bool

	// ExcludeMetadata determines whether <Meta> tags should be excluded while
	// encoding.
	ExcludeMetadata bool

	// Prefix is a string that appears at the start of each line in the
	// document. The prefix is added after each newline. Newlines are added
	// automatically when either Prefix or Indent is not empty.
	Prefix string

	// Indent is a string that indicates one level of indentation. A sequence of
	// indents will appear after the Prefix, an amount equal to the current
	// nesting depth in the markup.
	Indent string

	// NoDefaultIndent sets how Indent is interpreted when Indent is an empty
	// string. If false, an empty Indent will be interpreted as "\t".
	NoDefaultIndent bool

	// Suffix is a string that appears at the very end of the document. This
	// string is appended to the end of the file, after the root tag.
	Suffix string

	// ExcludeRoot determines whether the root tag should be excluded when
	// encoding. This can be combined with Prefix to write documents in-line.
	ExcludeRoot bool
}

// Encode formats root, writing the result to w.
func (e Encoder) Encode(w io.Writer, root *rbxfile.Root) (warn, err error) {
	codec := robloxCodec{
		ExcludeReferent: e.ExcludeReferent,
		ExcludeExternal: e.ExcludeExternal,
		ExcludeMetadata: e.ExcludeMetadata,
	}
	document, err := codec.Encode(root)
	if err != nil {
		return document.Warnings.Return(), fmt.Errorf("error encoding data: %w", err)
	}
	document.Prefix = e.Prefix
	if e.Indent == "" && !e.NoDefaultIndent {
		document.Indent = "\t"
	} else {
		document.Indent = e.Indent
	}
	document.Suffix = e.Suffix
	document.ExcludeRoot = e.ExcludeRoot
	if _, err = document.WriteTo(w); err != nil {
		return document.Warnings.Return(), fmt.Errorf("error encoding format: %w", err)
	}
	return document.Warnings.Return(), nil
}
