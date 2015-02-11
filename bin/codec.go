package bin

import (
	"errors"
	"github.com/robloxapi/rbxdump"
	"github.com/robloxapi/rbxfile"
)

// RobloxCodec implements Decoder and Encoder to emulate Roblox's internal
// codec as closely as possible.
type RobloxCodec struct{}

func (RobloxCodec) Decode(model *FormatModel, api *rbxdump.API) (root *rbxfile.Root, err error) {
	return nil, errors.New("not implemented")
}
func (RobloxCodec) Encode(root *rbxfile.Root, api *rbxdump.API) (model *FormatModel, err error) {
	return nil, errors.New("not implemented")
}
