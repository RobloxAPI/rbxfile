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
	"errors"
	"fmt"
)

////////////////////////////////////////////////////////////////

// Root represents the root of an instance tree. Root is not itself an
// instance, but a container for multiple root instances.
type Root struct {
	// Instances contains root instances contained in the tree.
	Instances []*Instance

	// Metadata contains metadata about the tree.
	Metadata map[string]string
}

// Copy creates a copy of the root and its contents.
//
// A copied reference within the tree is resolved so that it points to the
// corresponding copy of the original referent. Copied references that point
// to an instance which isn't being copied will still point to the same
// instance.
func (root *Root) Copy() *Root {
	clone := &Root{
		Instances: make([]*Instance, len(root.Instances)),
	}

	refs := make(References)
	crefs := make(References)
	propRefs := make([]PropRef, 0, 8)
	for i, inst := range root.Instances {
		clone.Instances[i] = inst.clone(refs, crefs, &propRefs)
	}
	for _, propRef := range propRefs {
		if !crefs.Resolve(propRef) {
			// Refers to an instance outside the tree, try getting the
			// original referent.
			refs.Resolve(propRef)
		}
	}
	return clone
}

// Instance represents a single Roblox instance.
type Instance struct {
	// ClassName indicates the instance's type.
	ClassName string

	// Reference is a unique string used to refer to the instance from
	// elsewhere in the tree.
	Reference string

	// IsService indicates whether the instance should be treated as a
	// service.
	IsService bool

	// Properties is a map of properties of the instance. It maps the name of
	// the property to its current value.
	Properties map[string]Value

	// Children contains instances that are the children of the current
	// instance.
	Children []*Instance

	// The parent of the instance. Can be nil.
	parent *Instance
}

// NewInstance creates a new Instance of a given class, and an optional
// parent.
func NewInstance(className string, parent *Instance) *Instance {
	inst := &Instance{
		ClassName:  className,
		Reference:  GenerateReference(),
		Properties: make(map[string]Value, 0),
	}
	if parent != nil {
		parent.Children = append(parent.Children, inst)
		inst.parent = parent
	}
	return inst
}

func assertLoop(child, parent *Instance) error {
	if parent == child {
		return fmt.Errorf("attempt to set %s as its own parent", child.Name())
	}
	if parent != nil && parent.IsDescendantOf(child) {
		return errors.New("attempt to set parent would result in circular reference")
	}
	return nil
}

func (inst *Instance) addChild(child *Instance) {
	if child.parent != nil {
		child.parent.RemoveChild(child)
	}
	inst.Children = append(inst.Children, child)
	child.parent = inst
}

// AddChild appends a child instance to the instance's list of children. If
// the child has a parent, it is first removed. The parent of the child is set
// to the instance. An error is returned if the instance is a descendant of
// the child, or if the child is the instance itself.
func (inst *Instance) AddChild(child *Instance) error {
	if err := assertLoop(child, inst); err != nil {
		return err
	}
	inst.addChild(child)
	return nil
}

// AddChildAt inserts a child instance into the instance's list of children at
// a given position. If the child has a parent, it is first removed. The
// parent of the child is set to the instance. If the index is outside the
// bounds of the list, then it is constrained. An error is returned if the
// instance is a descendant of the child, or if the child is the instance
// itself.
func (inst *Instance) AddChildAt(index int, child *Instance) error {
	if err := assertLoop(child, inst); err != nil {
		return err
	}
	if index < 0 {
		index = 0
	} else if index >= len(inst.Children) {
		inst.addChild(child)
		return nil
	}
	if child.parent != nil {
		child.parent.RemoveChild(child)
	}
	inst.Children = append(inst.Children, nil)
	copy(inst.Children[index+1:], inst.Children[index:])
	inst.Children[index] = child
	child.parent = inst
	return nil
}

func (inst *Instance) removeChildAt(index int) (child *Instance) {
	child = inst.Children[index]
	child.parent = nil
	copy(inst.Children[index:], inst.Children[index+1:])
	inst.Children[len(inst.Children)-1] = nil
	inst.Children = inst.Children[:len(inst.Children)-1]
	return child
}

// RemoveChild removes a child instance from the instance's list of children.
// The parent of the child is set to nil. Returns the removed child.
func (inst *Instance) RemoveChild(child *Instance) *Instance {
	for index, c := range inst.Children {
		if c == child {
			return inst.removeChildAt(index)
		}
	}
	return nil
}

// RemoveChildAt removes the child at a given position from the instance's
// list of children. The parent of the child is set to nil. If the index is
// outside the bounds of the list, then no children are removed. Returns the
// removed child.
func (inst *Instance) RemoveChildAt(index int) *Instance {
	if index < 0 || index >= len(inst.Children) {
		return nil
	}
	return inst.removeChildAt(index)
}

// RemoveAll remove every child from the instance. The parent of each child is
// set to nil.
func (inst *Instance) RemoveAll() {
	for i, child := range inst.Children {
		child.parent = nil
		inst.Children[i] = nil
	}
	inst.Children = inst.Children[:0]
}

// Parent returns the parent of the instance. Can return nil if the instance
// has no parent.
func (inst *Instance) Parent() *Instance {
	return inst.parent
}

// SetParent sets the parent of the instance, removing itself from the
// children of the old parent, and adding itself as a child of the new parent.
// The parent can be set to nil. An error is returned if the parent is a
// descendant of the instance, or if the parent is the instance itself. If the
// new parent is the same as the old parent, then the position of the instance
// in the parent's children is unchanged.
func (inst *Instance) SetParent(parent *Instance) error {
	if inst.parent == parent {
		return nil
	}
	if err := assertLoop(inst, parent); err != nil {
		return err
	}
	if inst.parent != nil {
		inst.parent.RemoveChild(inst)
	}
	if parent != nil {
		parent.addChild(inst)
	}
	return nil
}

func (inst *Instance) clone(refs, crefs References, propRefs *[]PropRef) *Instance {
	clone := &Instance{
		ClassName:  inst.ClassName,
		Reference:  refs.Get(inst),
		IsService:  inst.IsService,
		Children:   make([]*Instance, len(inst.Children)),
		Properties: make(map[string]Value, len(inst.Properties)),
	}
	crefs[clone.Reference] = clone
	for name, value := range inst.Properties {
		if value, ok := value.(ValueReference); ok {
			*propRefs = append(*propRefs, PropRef{
				Instance:  clone,
				Property:  name,
				Reference: refs.Get(value.Instance),
			})
			continue
		}
		clone.Properties[name] = value.Copy()
	}
	for i, child := range inst.Children {
		c := child.clone(refs, crefs, propRefs)
		clone.Children[i] = c
		c.parent = clone
	}
	return clone
}

// Clone returns a copy of the instance. Each property and all descendants are
// copied as well. Unlike Roblox's implementation, the Archivable property is
// ignored.
//
// A copied reference within the tree is resolved so that it points to the
// corresponding copy of the original referent. Copied references that point
// to an instance which isn't being copied will still point to the same
// instance.
func (inst *Instance) Clone() *Instance {
	refs := make(References)
	crefs := make(References)
	propRefs := make([]PropRef, 0, 8)
	clone := inst.clone(refs, crefs, &propRefs)
	for _, propRef := range propRefs {
		if !crefs.Resolve(propRef) {
			// Refers to an instance outside the tree, try getting the
			// original referent.
			refs.Resolve(propRef)
		}
	}
	return clone
}

// FindFirstChild returns the first found child whose Name property matches
// the given name. Returns nil if no child was found. If recursive is true,
// then FindFirstChild will be called on descendants as well.
func (inst *Instance) FindFirstChild(name string, recursive bool) *Instance {
	for _, child := range inst.Children {
		if child.Name() == name {
			return child
		}
	}

	if recursive {
		for _, child := range inst.Children {
			if desc := child.FindFirstChild(name, true); desc != nil {
				return desc
			}
		}
	}

	return nil
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
