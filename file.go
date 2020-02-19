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

// Root represents the root of an instance tree. Root is not itself an
// instance, but a container for multiple root instances.
type Root struct {
	// Instances contains root instances contained in the tree.
	Instances []*Instance

	// Metadata contains metadata about the tree.
	Metadata map[string]string
}

// NewRoot returns a new initialized Root.
func NewRoot() *Root {
	return &Root{
		Instances: []*Instance{},
		Metadata:  map[string]string{},
	}
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
		clone.Instances[i] = inst.copy(refs, crefs, &propRefs)
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
	// instance. The user must take care not to introduce circular references.
	Children []*Instance
}

// NewInstance creates a new Instance of a given class, and an optional
// parent.
func NewInstance(className string) *Instance {
	inst := &Instance{
		ClassName:  className,
		Reference:  GenerateReference(),
		Properties: make(map[string]Value, 0),
	}
	return inst
}

// copy returns a deep copy of the instance while managing references.
func (inst *Instance) copy(refs, crefs References, propRefs *[]PropRef) *Instance {
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
		c := child.copy(refs, crefs, propRefs)
		clone.Children[i] = c
	}
	return clone
}

// Copy returns a deep copy of the instance. Each property and all descendants
// are copied.
//
// A copied reference within the tree is resolved so that it points to the
// corresponding copy of the original referent. Copied references that point
// to an instance which isn't being copied will still point to the same
// instance.
func (inst *Instance) Copy() *Instance {
	refs := make(References)
	crefs := make(References)
	propRefs := make([]PropRef, 0, 8)
	clone := inst.copy(refs, crefs, &propRefs)
	for _, propRef := range propRefs {
		if !crefs.Resolve(propRef) {
			// Refers to an instance outside the tree, try getting the
			// original referent.
			refs.Resolve(propRef)
		}
	}
	return clone
}
