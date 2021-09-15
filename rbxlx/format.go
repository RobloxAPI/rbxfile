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
	if _, err = document.WriteTo(w); err != nil {
		return document.Warnings.Return(), fmt.Errorf("error encoding format: %w", err)
	}
	return document.Warnings.Return(), nil
}
