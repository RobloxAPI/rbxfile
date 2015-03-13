package rbxfile_test

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/robloxapi/rbxdump"
	"github.com/robloxapi/rbxfile"
	"io"
	"regexp"
	"testing"
)

// Instance Tests

func TestNewInstance(t *testing.T) {
	inst := rbxfile.NewInstance("Part", nil)
	if inst.ClassName != "Part" {
		t.Errorf("got ClassName %q, expected %q", inst.ClassName, "Part")
	}
	if ok, _ := regexp.Match("^RBX[0-9A-F]{32}$", inst.Reference); !ok {
		t.Errorf("unexpected value for generated reference")
	}

	child := rbxfile.NewInstance("IntValue", inst)
	if child.ClassName != "IntValue" {
		t.Errorf("got ClassName %q, expected %q", child.ClassName, "IntValue")
	}
	if ok, _ := regexp.Match("^RBX[0-9A-F]{32}$", child.Reference); !ok {
		t.Errorf("unexpected value for generated reference")
	}
	if child.Parent() != inst {
		t.Errorf("parent of child is not inst")
	}
}

func namedInst(className string, parent *rbxfile.Instance) *rbxfile.Instance {
	inst := rbxfile.NewInstance(className, parent)
	inst.SetName(className)
	return inst
}

type h struct {
	a, b   *rbxfile.Instance
	ar, dr bool
}

func testAncestry(t *testing.T, groups ...h) {
	for _, g := range groups {
		if r := g.a.IsAncestorOf(g.b); r != g.ar {
			t.Errorf("%s.IsAncestorOf(%s) returned %t when %t was expected", g.a, g.b, r, g.ar)
		}
		if r := g.a.IsDescendantOf(g.b); r != g.dr {
			t.Errorf("%s.IsDescendantOf(%s) returned %t when %t was expected", g.a, g.b, r, g.dr)
		}
	}
}

func TestInstanceHierarchy(t *testing.T) {
	parent := namedInst("Parent", nil)
	inst := namedInst("Instance", nil)
	sibling := namedInst("Sibling", parent)
	child := namedInst("Child", inst)
	desc := namedInst("Descendant", child)

	if inst.Parent() != nil {
		t.Error("expected nil parent")
	}

	if err := inst.SetParent(inst); err == nil {
		t.Error("no error on setting parent to self")
	}

	if err := inst.SetParent(child); err == nil {
		t.Error("no error on setting parent to child")
	}

	if err := inst.SetParent(desc); err == nil {
		t.Error("no error on setting parent to descendant")
	}

	if err := inst.SetParent(parent); err != nil {
		t.Error("failed to set parent:", err)
	}

	if inst.Parent() != parent {
		t.Error("unexpected parent")
	}

	if err := inst.SetParent(parent); err != nil {
		t.Error("error on setting same parent:", err)
	}

	testAncestry(t,
		h{parent, nil, false, false},
		h{parent, parent, false, false},
		h{parent, sibling, true, false},
		h{parent, inst, true, false},
		h{parent, child, true, false},
		h{parent, desc, true, false},

		h{sibling, nil, false, false},
		h{sibling, parent, false, true},
		h{sibling, sibling, false, false},
		h{sibling, inst, false, false},
		h{sibling, child, false, false},
		h{sibling, desc, false, false},

		h{inst, nil, false, false},
		h{inst, parent, false, true},
		h{inst, sibling, false, false},
		h{inst, inst, false, false},
		h{inst, child, true, false},
		h{inst, desc, true, false},

		h{child, nil, false, false},
		h{child, parent, false, true},
		h{child, sibling, false, false},
		h{child, inst, false, true},
		h{child, child, false, false},
		h{child, desc, true, false},

		h{desc, nil, false, false},
		h{desc, parent, false, true},
		h{desc, sibling, false, false},
		h{desc, inst, false, true},
		h{desc, child, false, true},
		h{desc, desc, false, false},
	)

	if err := sibling.SetParent(nil); err != nil {
		t.Error("failed to set parent:", err)
	}

	if sibling.Parent() != nil {
		t.Error("expected nil parent")
	}

	if err := sibling.SetParent(parent); err != nil {
		t.Error("failed to set parent:", err)
	}

	if sibling.Parent() != parent {
		t.Error("unexpected parent")
	}
}

func TestInstance_ClearAllChildren(t *testing.T) {
	inst := rbxfile.NewInstance("Instance", nil)
	for i := 0; i < 10; i++ {
		rbxfile.NewInstance("Child", inst)
	}

	if n := len(inst.GetChildren()); n != 10 {
		t.Fatalf("expected 10 children, got %d", n)
	}

	inst.ClearAllChildren()

	if n := len(inst.GetChildren()); n != 0 {
		t.Fatalf("expected %d children, got %d", 0, n)
	}
}

func TestInstance_Clone(t *testing.T) {
	inst := rbxfile.NewInstance("Instance", nil)
	inst.SetName("InstanceName")
	inst.Properties["Position"] = rbxfile.ValueVector3{1, 2, 3}

	child := rbxfile.NewInstance("Child", inst)
	child.SetName("ChildName")
	child.Properties["Size"] = rbxfile.ValueVector3{4, 5, 6}

	cinst := inst.Clone()

	if cinst.ClassName != inst.ClassName {
		t.Error("cloned ClassName does not equal original")
	}
	if cinst.Parent() != nil {
		t.Error("expected nil clone parent")
	}

	if cinst.Name() != inst.Name() {
		t.Error("cloned Name property does not equal original")
	}
	if cinst.Properties["Position"] != inst.Properties["Position"] {
		t.Error("cloned Position property does not equal original")
	}

	var cchild *rbxfile.Instance
	if cchildren := cinst.GetChildren(); len(cchildren) != 1 {
		t.Fatalf("expected 1 child, got %d", len(cchildren))
	} else {
		cchild = cchildren[0]
	}

	if cchild.ClassName != child.ClassName {
		t.Error("cloned child ClassName does not equal original")
	}
	if cchild.Parent() != cinst {
		t.Error("clone child parent is not cloned inst")
	}

	if cchild.Name() != child.Name() {
		t.Error("cloned child Name property does not equal original")
	}
	if cchild.Properties["Size"] != child.Properties["Size"] {
		t.Error("cloned child Size property does not equal original")
	}
}

func TestInstance_FindFirstChild(t *testing.T) {
	inst := namedInst("Instance", nil)
	child0 := namedInst("Child", inst)
	desc00 := namedInst("Desc", child0)
	namedInst("Desc", child0)
	child1 := namedInst("Child", inst)
	namedInst("Desc", child1)
	desc11 := namedInst("Desc1", child1)
	namedInst("Desc", child1)

	if c := inst.FindFirstChild("DoesNotExist", false); c != nil {
		t.Error("found child that does not exist")
	}

	if c := inst.FindFirstChild("DoesNotExist", true); c != nil {
		t.Error("found descendant that does not exist (recursive)")
	}

	if c := inst.FindFirstChild("Child", false); c != child0 {
		t.Error("failed to get first child")
	}

	if c := inst.FindFirstChild("Child", true); c != child0 {
		t.Error("failed to get first child (recursive)")
	}

	if c := inst.FindFirstChild("Desc", false); c != nil {
		t.Error("expected nil result")
	}

	if c := inst.FindFirstChild("Desc", true); c != desc00 {
		t.Error("failed to get first descendant (recursive)")
	}

	if c := inst.FindFirstChild("Desc1", true); c != desc11 {
		t.Error("failed to get selected descendant (recursive)")
	}
}

func TestInstance_GetChildren(t *testing.T) {
	inst := rbxfile.NewInstance("Instance", nil)
	children := make([]*rbxfile.Instance, 4)
	for i := 0; i < len(children); i++ {
		children[i] = rbxfile.NewInstance("Child", inst)
	}

	ch := inst.GetChildren()
	if len(ch) != len(children) {
		t.Fatalf("expected %d children, got %d", len(children), len(ch))
	}

	for i, child := range ch {
		if child != children[i] {
			t.Errorf("unexpected child #%d from GetChildren", i)
		}
	}
}

func TestInstance_GetFullName(t *testing.T) {
	inst0 := namedInst("Grandparent", nil)
	inst1 := namedInst("Parent", inst0)
	inst2 := namedInst("Entity", inst1)
	inst3 := namedInst("Child", inst2)
	inst4 := namedInst("Grandchild", inst3)

	if name := inst4.GetFullName(); name != `Grandparent.Parent.Entity.Child.Grandchild` {
		t.Errorf("unexpected full name %q", name)
	}
}

func TestInstance_Remove(t *testing.T) {
	parent := rbxfile.NewInstance("Parent", nil)
	inst := rbxfile.NewInstance("Instance", parent)
	child := rbxfile.NewInstance("Child", inst)
	desc := rbxfile.NewInstance("Descendant", child)

	inst.Remove()

	if inst.Parent() != nil {
		t.Error("expected instance parent to be nil")
	}
	if child.Parent() != nil {
		t.Error("expected child parent to be nil")
	}
	if desc.Parent() != nil {
		t.Error("expected descendant parent to be nil")
	}
}

func TestInstance_Name(t *testing.T) {
	inst := rbxfile.NewInstance("Instance", nil)

	if inst.Name() != "" {
		t.Error("unexpected value returned from Name")
	}

	inst.SetName("Instance")

	if v, ok := inst.Properties["Name"]; !ok {
		t.Error("failed to set Name property")
	} else if v, ok := v.(rbxfile.ValueString); !ok {
		t.Error("expected ValueString type for Name property")
	} else if string(v) != "Instance" {
		t.Error("unexpected value of Name property")
	}

	if inst.Name() != "Instance" {
		t.Error("unexpected value returned from Name")
	}

	inst.SetName("")

	if v, ok := inst.Properties["Name"]; !ok {
		t.Error("expected Name property")
	} else if v, ok := v.(rbxfile.ValueString); !ok {
		t.Error("expected ValueString type for Name property")
	} else if string(v) != "" {
		t.Error("unexpected value of Name property")
	}

	if inst.Name() != "" {
		t.Error("unexpected value returned from Name")
	}
}

func TestInstance_String(t *testing.T) {
	inst := rbxfile.NewInstance("Instance", nil)

	if inst.String() != "Instance" {
		t.Error("unexpected value returned from String")
	}

	inst.SetName("InstanceName")

	if inst.String() != "InstanceName" {
		t.Error("unexpected value returned from String")
	}

	inst.SetName("")

	if inst.String() != "Instance" {
		t.Error("unexpected value returned from String")
	}
}

func TestInstance_GetSet(t *testing.T) {
	inst := rbxfile.NewInstance("Instance", nil)

	if inst.Get("Property") != nil {
		t.Error("unexpected value returned from Get")
	}

	inst.Set("Property", rbxfile.ValueString("Value"))

	if v, ok := inst.Properties["Property"]; !ok {
		t.Error("expected property")
	} else if v, ok := v.(rbxfile.ValueString); !ok {
		t.Error("expected ValueString type for property")
	} else if string(v) != "Value" {
		t.Error("unexpected value of property")
	}

	if v := inst.Get("Property"); v == nil {
		t.Error("expected property")
	} else if v, ok := v.(rbxfile.ValueString); !ok {
		t.Error("expected ValueString type for property")
	} else if string(v) != "Value" {
		t.Error("unexpected value of property")
	}

	inst.Set("Property", nil)

	if inst.Properties["Property"] != nil {
		t.Error("unexpected property")
	}

	if inst.Get("Property") != nil {
		t.Error("unexpected value returned from Get")
	}
}

// Format Tests

// Format implements rbxfile.Format so that this package can be registered
// when it is imported.
type format struct {
	name   string
	magic  string
	decode func(io.Reader, *rbxdump.API) (*rbxfile.Root, error)
	encode func(io.Writer, *rbxdump.API, *rbxfile.Root) error
}

func (f format) Name() string {
	return f.name
}

func (f format) Magic() string {
	return f.magic
}

func (f format) Decode(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
	return f.decode(r, api)
}

func (f format) Encode(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
	return f.encode(w, api, root)
}

func TestFormat(t *testing.T) {
	rbxfile.RegisterAPI(nil)

	rbxfile.RegisterFormat(format{
		name:  "test",
		magic: "test????signature",
		decode: func(r io.Reader, api *rbxdump.API) (root *rbxfile.Root, err error) {
			return nil, errors.New("decode success")
		},
		encode: func(w io.Writer, api *rbxdump.API, root *rbxfile.Root) (err error) {
			return errors.New("encode success")
		},
	})

	r := bytes.NewBufferString("testgoodsignature testcontent")
	if _, err := rbxfile.Decode(r); err == nil || err.Error() != "decode success" {
		t.Error("unexpected result from Decode:", err)
	}

	r = bytes.NewBufferString("testbadsig testcontent")
	if _, err := rbxfile.Decode(r); err != rbxfile.ErrFormat {
		t.Error("unexpected result from Decode:", err)
	}

	buf := bufio.NewReader(bytes.NewBufferString("testbuffsignature testcontent"))
	if _, err := rbxfile.Decode(buf); err == nil || err.Error() != "decode success" {
		t.Error("unexpected result from Decode:", err)
	}

	w := new(bytes.Buffer)
	if err := rbxfile.Encode(w, "test", nil); err == nil || err.Error() != "encode success" {
		t.Error("unexpected result from Encode:", err)
	}

	if err := rbxfile.Encode(w, "badformat", nil); err != rbxfile.ErrFormat {
		t.Error("unexpected result from Encode:", err)
	}
}
