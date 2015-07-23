package xml

import (
	"github.com/robloxapi/rbxdump"
	"github.com/robloxapi/rbxfile"
	"io"
)

type Format struct{}

func (Format) Name() string {
	return "xml"
}

func (Format) Magic() string {
	return "<roblox "
}

func (f Format) Decode(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
	return nil, nil
}
func (f Format) Encode(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
	return nil
}
