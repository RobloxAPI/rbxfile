// Package rbxfile implements a library for decoding and encoding Roblox
// instance file formats.
//
// This package can be used to manipulate Roblox instance trees outside of the
// Roblox client. Such data structures begin with a Root struct. A Root
// contains a list of child Instances, which in turn contain more child
// Instances, and so on, forming a tree of Instances. These Instances can be
// accessed and manipulated using an API similar to that of Roblox.
//
// Each Instance also has a set of "properties". Each property has a specific
// value of a certain type. Every available type is prefixed with "Value", and
// implements the Value interface.
package rbxfile

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/robloxapi/rbxdump"
	"github.com/satori/go.uuid"
	"io"
)

////////////////////////////////////////////////////////////////

// Root represents the root of an instance tree. Root is not itself an
// instance, but a container for multiple root instances.
type Root struct {
	// Instances contains root instances contained in the tree.
	Instances []*Instance
}

// Instance represents a single Roblox instance.
type Instance struct {
	// ClassName indicates the instance's type.
	ClassName string

	// Properties is a map of properties of the instance. It maps the name of
	// the property to its current value.
	Properties map[string]Value

	// Reference is a unique string used to refer to the instance from
	// elsewhere in the tree.
	Reference []byte

	// IsService indicates whether the instance should be treated as a
	// service.
	IsService bool

	// Contains instances that are the children of the current instance.
	children []*Instance

	// The parent of the instance. Can be nil.
	parent *Instance
}

// NewInstance creates a new Instance of a given class, and an optional
// parent.
func NewInstance(className string, parent *Instance) *Instance {
	inst := &Instance{
		ClassName:  className,
		Properties: make(map[string]Value, 0),
	}

	ref := uuid.NewV4()
	inst.Reference = make([]byte, 3+hex.EncodedLen(len(ref)))
	copy(inst.Reference[:3], []byte("RBX"))
	hex.Encode(inst.Reference[3:], ref.Bytes())
	inst.Reference = bytes.ToUpper(inst.Reference)

	if parent != nil {
		inst.SetParent(parent)
	}

	return inst
}

func (inst *Instance) addChild(child *Instance) bool {
	if child == nil {
		return false
	}

	for _, ch := range inst.children {
		if ch == child {
			return false
		}
	}

	inst.children = append(inst.children, child)

	return true
}

func (inst *Instance) removeChild(child *Instance) bool {
	if child == nil {
		return false
	}

	for i, ch := range inst.children {
		if ch == child {
			inst.children[i] = nil
			inst.children = append(inst.children[:i], inst.children[i+1:]...)
			return true
		}
	}

	return false
}

// Parent returns the parent of the instance. Can return nil if the instance
// has no parent.
func (inst *Instance) Parent() *Instance {
	return inst.parent
}

// SetParent sets the parent of the instance. The parent can be set to nil.
// The function will error if the parent is a descendant of the instance.
func (inst *Instance) SetParent(parent *Instance) error {
	if inst.parent == parent {
		return nil
	}

	if parent == inst {
		return fmt.Errorf("attempt to set %s as its own parent", inst.Name())
	}

	if parent != nil && parent.IsDescendantOf(inst) {
		return errors.New("attempt to set parent would result in circular reference")
	}

	if inst.parent != nil {
		inst.parent.removeChild(inst)
	}

	inst.parent = parent

	if parent != nil {
		parent.addChild(inst)
	}

	return nil
}

// ClearAllChildren sets the Parent of each of the children of the instance to
// nil.
func (inst *Instance) ClearAllChildren() {
	// Note: Roblox's ClearAllChildren traverses children in reverse, calling
	// Remove on each child. Remove removes children in order, also calling
	// Remove on each of them.
	for len(inst.children) > 0 {
		inst.children[len(inst.children)].Remove()
	}
}

// Clone returns a copy of the instance. Each property and all descendants are
// copied as well. Unlike Roblox's implementation, the Archivable property is
// ignored.
func (inst *Instance) Clone() *Instance {
	clone := NewInstance(inst.ClassName, nil)

	for name, value := range inst.Properties {
		clone.Properties[name] = value.Copy()
	}

	for _, child := range inst.children {
		child.Clone().SetParent(clone)
	}

	return clone
}

// FindFirstChild returns the first found child whose Name property matches
// the given name. Returns nil if no child was found. If recursive is true,
// then FindFirstChild will be called on descendants as well.
func (inst *Instance) FindFirstChild(name string, recursive bool) *Instance {
	for _, child := range inst.children {
		if child.Name() == name {
			return child
		}
	}

	if recursive {
		for _, child := range inst.children {
			desc := child.FindFirstChild(name, true)
			if desc != nil {
				return desc
			}
		}
	}

	return nil
}

// GetChildren returns a list of children of the instance.
func (inst *Instance) GetChildren() []*Instance {
	list := make([]*Instance, len(inst.children))
	copy(list, inst.children)
	return list
}

// GetFullName returns the "full" name of the instance, which is the combined
// names of the instance and every ancestor, separated by a `.` character.
func (inst *Instance) GetFullName() string {
	// Note: Roblox's GetFullName stops at the first ancestor that is a
	// ServiceProvider. Since recreating this behavior would require
	// information about the class hierarchy, this implementation simply
	// includes every ancestor.

	names := make([]string, 0, 8)

	object := inst
	for object != nil {
		names = append(names, object.Name())
		object = object.Parent()
	}

	full := make([]byte, 0, 64)
	for i := len(names); i > 0; i-- {
		full = append(full, []byte(names[i])...)
		full = append(full, '.')
	}
	full = append(full, []byte(names[0])...)

	return string(full)
}

// IsAncestorOf returns whether the instance is the ancestor of another
// instance.
func (inst *Instance) IsAncestorOf(descendant *Instance) bool {
	if descendant != nil {
		return descendant.IsDescendantOf(inst)
	}
	return false
}

// IsDescendantOf returns whether the instance is the descendant of another
// instance.
func (inst *Instance) IsDescendantOf(ancestor *Instance) bool {
	parent := inst.Parent()
	for parent != nil {
		if parent == ancestor {
			return true
		}
		parent = parent.Parent()
	}
	return false
}

// Remove sets the parent of the instance to nil, then calls Remove on each of
// the instance's children.
func (inst *Instance) Remove() {
	inst.SetParent(nil)

	// Roblox's implementation removes children in order.
	for len(inst.children) > 0 {
		inst.children[0].Remove()
	}
}

// Name returns the Name property of the instance, or an empty string if it is
// invalid or not defined.
func (inst *Instance) Name() string {
	iname, ok := inst.Properties["Name"]
	if !ok {
		return ""
	}

	return string(iname.(ValueString))
}

// SetName sets the Name property of the instance.
func (inst *Instance) SetName(name string) {
	inst.Properties["Name"] = ValueString(name)
}

// Get returns the value of a property in the instance. The value will be nil
// if the property is not defined.
func (inst *Instance) Get(property string) (value Value) {
	return inst.Properties[property]
}

// Set sets the value of a property in the instance. If value is nil, then the
// value will be deleted from the Properties map.
func (inst *Instance) Set(property string, value Value) {
	if value == nil {
		delete(inst.Properties, property)
	} else {
		inst.Properties[property] = value
	}
}

////////////////////////////////////////////////////////////////

// Format encodes and decodes a single file format for a data structure.
type Format interface {
	// Name returns the name of the format.
	Name() string

	// Magic returns a magic prefix that identifies the format. The magic
	// string can contain "?" wildcards that each match any one byte.
	Magic() string

	// Decode decodes data from r into a data structure. API is an API dump
	// that can be used while decoding, and may be nil.
	Decode(r io.Reader, api *rbxdump.API) (root *Root, err error)

	// Encode encodes a data structure into w. API is a Roblox API that can be
	// used while encoding, and may be nil.
	Encode(w io.Writer, api *rbxdump.API, root *Root) (err error)
}

var formats []Format

// RegisterFormat registers a file format for use by Codec.
func RegisterFormat(format Format) {
	formats = append(formats, format)
}

////////////////////////////////////////////////////////////////

var ErrFormat = errors.New("unknown format")

// Codec encodes and decodes Roblox files using registered formats.
type Codec struct {
	// API is an API structure that can be used by formats to ensure that data
	// is encoded and decoded correctly.
	API *rbxdump.API
}

func match(magic string, b []byte) bool {
	if len(magic) != len(b) {
		return false
	}
	for i, c := range b {
		if magic[i] != c && magic[i] != '?' {
			return false
		}
	}
	return true
}

// Decode attempts to determine and decode the format of the underlying data
// stream in `r` by reading the header. Only registered formats are detected.
//
// Returns ErrFormat if the format could not be detected.
func (c *Codec) Decode(r io.Reader) (root *Root, err error) {
	var buf *bufio.Reader
	if br, ok := r.(*bufio.Reader); ok {
		buf = br
	} else {
		buf = bufio.NewReader(r)
	}

	var format Format
	for _, f := range formats {
		magic := f.Magic()
		header, err := buf.Peek(len(magic))
		if err == nil && match(magic, header) {
			format = f
		}
	}
	if format == nil {
		return nil, ErrFormat
	}

	return format.Decode(buf, c.API)
}

// Encode attempts to encode a data structure to a given format. The fmt
// argument should match the name given by the format's Name() method. Only
// registered formats can be encoded to.
//
// Returns ErrFormat if the given format is not registered.
func (c *Codec) Encode(w io.Writer, fmt string, root *Root) (err error) {
	var format Format
	for _, f := range formats {
		if fmt == f.Name() {
			format = f
			return
		}
	}
	if format == nil {
		return ErrFormat
	}

	return format.Encode(w, c.API, root)
}

////////////////////////////////////////////////////////////////

// DefaultCodec is the Codec used by Encode and Decode.
var DefaultCodec = &Codec{}

// RegisterAPI registers an API structure to be used by Encode and Decode.
func RegisterAPI(api *rbxdump.API) {
	DefaultCodec.API = api
}

// Decode attempts to determine and decode the format of the underlying data
// stream in `r` by reading the header. Only registered formats are detected.
//
// Returns ErrFormat if the format could not be detected.
func Decode(r io.Reader) (root *Root, err error) {
	return DefaultCodec.Decode(r)
}

// Encode attempts to encode a data structure to a given format. The fmt
// argument should match the name given by the format's Name() method. Only
// registered formats can be encoded to.
//
// Returns ErrFormat if the given format is not registered.
func Encode(w io.Writer, fmt string, root *Root) (err error) {
	return DefaultCodec.Encode(w, fmt, root)
}
