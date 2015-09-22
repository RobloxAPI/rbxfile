// The rbxfile package handles the decoding, encoding, and manipulation of
// Roblox instance data structures.
//
// This package can be used to manipulate Roblox instance trees outside of the
// Roblox client. Such data structures begin with a Root struct. A Root
// contains a list of child Instances, which in turn contain more child
// Instances, and so on, forming a tree of Instances. These Instances can be
// accessed and manipulated using an API similar to that of Roblox.
//
// Each Instance also has a set of "properties". Each property has a specific
// value of a certain type. Every available type implements the Value
// interface, and is prefixed with "Value".
//
// Root structures can be decoded from and encoded to various formats,
// including Roblox's native file formats. The two sub-packages "bin" and
// "xml" provide formats for Roblox's binary and XML formats. Root structures
// can also be encoded and decoded with the "json" package.
//
// Besides decoding from a format, root structures can also be created
// manually. The best way to do this is through the "declare" sub-package,
// which provides an easy way to generate root structures.
package rbxfile

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
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

func (inst *Instance) addChild(child *Instance) {
	inst.children = append(inst.children, child)
}

func (inst *Instance) removeChild(child *Instance) {
	for i, ch := range inst.children {
		if ch == child {
			inst.children[i] = nil
			inst.children = append(inst.children[:i], inst.children[i+1:]...)
		}
	}
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
		inst.children[len(inst.children)-1].Remove()
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
	for i := len(names) - 1; i > 0; i-- {
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

	name, _ := iname.(ValueString)
	return string(name)
}

// String implements the fmt.Stringer interface by returning the Name of the
// instance, or the ClassName if Name isn't defined.
func (inst *Instance) String() string {
	iname, ok := inst.Properties["Name"]
	if !ok {
		return inst.ClassName
	}

	name, _ := iname.(ValueString)
	if string(name) == "" {
		return inst.ClassName
	}

	return string(name)
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
