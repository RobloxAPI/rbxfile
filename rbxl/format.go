// Package rbxl implements a decoder and encoder for Roblox's binary file
// format.
package rbxl

// Mode indicates how the codec formats data.
type Mode uint8

const (
	Place Mode = iota // Data is handled as a Roblox place (RBXL) file.
	Model             // Data is handled as a Roblox model (RBXM) file.
)
