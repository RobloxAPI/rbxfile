// Package rbxl implements a decoder and encoder for Roblox's binary file
// format.
//
// The easiest way to decode and encode files is through the functions
// DeserializePlace, SerializePlace, DeserializeModel, and SerializeModel.
// These decode and encode directly between byte streams and Root structures
// specified by the rbxfile package.
package rbxl

import (
	"io"

	"github.com/robloxapi/rbxfile"
)

// Deserialize decodes data from r into a Root structure using the default
// decoder. Data is interpreted as a Roblox place file.
func DeserializePlace(r io.Reader) (root *rbxfile.Root, err error) {
	return Decoder{Mode: ModePlace}.Decode(r)
}

// Serialize encodes data from a Root structure to w using the default
// encoder. Data is interpreted as a Roblox place file.
func SerializePlace(w io.Writer, root *rbxfile.Root) (err error) {
	return Encoder{Mode: ModePlace}.Encode(w, root)
}

// Deserialize decodes data from r into a Root structure using the default
// decoder. Data is interpreted as a Roblox model file.
func DeserializeModel(r io.Reader) (root *rbxfile.Root, err error) {
	return Decoder{Mode: ModeModel}.Decode(r)
}

// Serialize encodes data from a Root structure to w using the default
// encoder. Data is interpreted as a Roblox model file.
func SerializeModel(w io.Writer, root *rbxfile.Root) (err error) {
	return Encoder{Mode: ModeModel}.Encode(w, root)
}
