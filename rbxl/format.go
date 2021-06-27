// Package rbxl implements a decoder and encoder for Roblox's binary file
// format.
//
// The easiest way to decode and encode files is through the functions
// DeserializePlace, SerializePlace, DeserializeModel, and SerializeModel.
// These decode and encode directly between byte streams and Root structures
// specified by the rbxfile package.
package rbxl

// Mode indicates how the codec formats data.
type Mode uint8

const (
	Place Mode = iota // Data is handled as a Roblox place (RBXL) file.
	Model             // Data is handled as a Roblox model (RBXM) file.
)
