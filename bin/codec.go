package bin

import (
	"errors"
	"github.com/robloxapi/rbxdump"
	"github.com/robloxapi/rbxfile"
)

// Mode indicates how RobloxCodec should interpret data.
type Mode uint8

const (
	ModePlace Mode = iota // Data is decoded and encoded as a Roblox place (RBXL) file.
	ModeModel             // Data is decoded and encoded as a Roblox model (RBXM) file.
)

// RobloxCodec implements Decoder and Encoder to emulate Roblox's internal
// codec as closely as possible.
type RobloxCodec struct {
	Mode Mode
}

func (RobloxCodec) Decode(model *FormatModel, api *rbxdump.API) (root *rbxfile.Root, err error) {
	return nil, errors.New("not implemented")
}
func (RobloxCodec) Encode(root *rbxfile.Root, api *rbxdump.API) (model *FormatModel, err error) {
	return nil, errors.New("not implemented")
}
