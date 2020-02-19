// +build ignore

package rbxfile

import (
	"bytes"
	"regexp"
	"strconv"
	"testing"
)

// Root Tests

func TestRootCopy(t *testing.T) {
	r := &Root{
		Instances: []*Instance{
			NewInstance("ReferToSelf", nil),
			NewInstance("ReferToSibling", nil),
			NewInstance("ReferToOutside", nil),
			NewInstance("HasChild", nil),
		},
	}
	child := NewInstance("Child", r.Instances[3])
	child.Set("TestByteCopy", ValueString("hello world"))
	outside := NewInstance("Outside", nil)
	r.Instances[0].Set("Reference", ValueReference{Instance: r.Instances[0]})
	r.Instances[1].Set("Reference", ValueReference{Instance: r.Instances[0]})
	r.Instances[2].Set("Reference", ValueReference{Instance: outside})

	rc := r.Copy()

	// Test number of instances.
	if i, j := len(r.Instances), len(rc.Instances); i != j {
		t.Errorf("mismatched number of instances (expected %d, got %d)", i, j)
	}
	for i := 0; i < len(r.Instances); i++ {
		if a, b := r.Instances[i].ClassName, rc.Instances[i].ClassName; a != b {
			t.Errorf("mismatched instance %d (expected %s, got %s)", a, b)
		}
		if r.Instances[i] == rc.Instances[i] {
			t.Errorf("instance %d in copy equals instance in root", i)
		}
	}
	// Test refer to self.
	if v, ok := rc.Instances[0].Get("Reference").(ValueReference); !ok || v.Instance != rc.Instances[0] {
		str := "<nil>"
		if v.Instance != nil {
			str = v.Instance.ClassName
		}
		t.Errorf("ReferToSelf failed (expected ReferToSelf, got %s)", str)
	}
	// Test refer to instance in tree.
	if v, ok := rc.Instances[1].Get("Reference").(ValueReference); !ok || v.Instance != rc.Instances[0] {
		str := "<nil>"
		if v.Instance != nil {
			str = v.Instance.ClassName
		}
		t.Errorf("ReferToSibling failed (expected ReferToSelf, got %s)", str)
	}
	// Test refer to instance outside tree.
	if v, _ := rc.Instances[2].Get("Reference").(ValueReference); v.Instance != outside {
		t.Errorf("ReferToOutside referent in copy does not equal referent in root")
	}
	// Test number of children.
	if i, j := len(r.Instances[3].Children), len(rc.Instances[3].Children); i != j {
		t.Errorf("mismatched number of children (expected %d, got %d)", i, j)
	}
	// Test children.
	if a, b := r.Instances[3].Children[0], rc.Instances[3].Children[0]; a == b {
		t.Errorf("child in copy equals child in root")
	} else {
		av := a.Get("TestByteCopy").(ValueString)
		bv, ok := b.Get("TestByteCopy").(ValueString)
		if !ok {
			t.Errorf("TestByteCopy: property failed to copy")
		}
		if !bytes.Equal([]byte(av), []byte(bv)) {
			t.Errorf("TestByteCopy: content of bytes not equal (got %v)", bv)
		}
		if av[0] = av[0] + 1; bv[0] == av[0] {
			t.Errorf("TestByteCopy: slices not copied")
		}
	}

	// Test parent of root instance.
	r = &Root{Instances: []*Instance{child}}
	rc = r.Copy()
	if rc.Instances[0].Parent() != nil {
		t.Errorf("instance has non-nil parent")
	}
}

// Instance Tests

func TestNewInstance(t *testing.T) {
	inst := NewInstance("Part", nil)
	if inst.ClassName != "Part" {
		t.Errorf("got ClassName %q, expected %q", inst.ClassName, "Part")
	}
	if ok, _ := regexp.MatchString("^RBX[0-9A-F]{32}$", inst.Reference); !ok {
		t.Errorf("unexpected value for generated reference")
	}

	child := NewInstance("IntValue", inst)
	if child.ClassName != "IntValue" {
		t.Errorf("got ClassName %q, expected %q", child.ClassName, "IntValue")
	}
	if ok, _ := regexp.MatchString("^RBX[0-9A-F]{32}$", child.Reference); !ok {
		t.Errorf("unexpected value for generated reference")
	}
	if child.Parent() != inst {
		t.Errorf("parent of child is not inst")
	}
	for _, c := range inst.Children {
		if c == child {
			goto foundChild
		}
	}
	t.Errorf("child not found in parent")
foundChild:
}

func namedInst(className string, parent *Instance) *Instance {
	inst := NewInstance(className, parent)
	inst.SetName(className)
	return inst
}

type treeTest struct {
	// Compare a to b.
	a, b *Instance
	// Expected results.
	ar, dr bool
}

func testAncestry(t *testing.T, groups ...treeTest) {
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
		treeTest{parent, nil, false, false},
		treeTest{parent, parent, false, false},
		treeTest{parent, sibling, true, false},
		treeTest{parent, inst, true, false},
		treeTest{parent, child, true, false},
		treeTest{parent, desc, true, false},

		treeTest{sibling, nil, false, false},
		treeTest{sibling, parent, false, true},
		treeTest{sibling, sibling, false, false},
		treeTest{sibling, inst, false, false},
		treeTest{sibling, child, false, false},
		treeTest{sibling, desc, false, false},

		treeTest{inst, nil, false, false},
		treeTest{inst, parent, false, true},
		treeTest{inst, sibling, false, false},
		treeTest{inst, inst, false, false},
		treeTest{inst, child, true, false},
		treeTest{inst, desc, true, false},

		treeTest{child, nil, false, false},
		treeTest{child, parent, false, true},
		treeTest{child, sibling, false, false},
		treeTest{child, inst, false, true},
		treeTest{child, child, false, false},
		treeTest{child, desc, true, false},

		treeTest{desc, nil, false, false},
		treeTest{desc, parent, false, true},
		treeTest{desc, sibling, false, false},
		treeTest{desc, inst, false, true},
		treeTest{desc, child, false, true},
		treeTest{desc, desc, false, false},
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

func TestInstance_AddChild(t *testing.T) {
	parent := namedInst("Parent", nil)
	inst := namedInst("Instance", nil)
	sibling := namedInst("Sibling", parent)
	child := namedInst("Child", inst)
	desc := namedInst("Descendant", child)

	if inst.Parent() != nil {
		t.Error("expected nil parent")
	}
	if err := inst.AddChild(inst); err == nil {
		t.Error("no error on adding self")
	}
	if err := child.AddChild(inst); err == nil {
		t.Error("no error on adding parent to child")
	}
	if err := desc.AddChild(inst); err == nil {
		t.Error("no error on adding parent to descendant")
	}
	if err := parent.AddChild(inst); err != nil {
		t.Error("failed add child:", err)
	}
	if inst.Parent() != parent {
		t.Error("unexpected parent")
	}
	if n := len(parent.Children); n != 2 {
		t.Error("unexpected length of children (expected 2, got %d):", n)
	}
	if parent.Children[0] != sibling {
		t.Error("unexpected sibling")
	}
	if parent.Children[1] != inst {
		t.Error("unexpected order of child")
	}
	if err := parent.AddChild(inst); err != nil {
		t.Error("error on adding same child:", err)
	}
	parent.AddChild(sibling)
	if parent.Children[0] != inst {
		t.Error("unexpected order of child")
	}
	if parent.Children[1] != sibling {
		t.Error("unexpected order of child")
	}
}

func TestInstance_AddChildAt(t *testing.T) {
	parent := namedInst("Parent", nil)
	inst := namedInst("Instance", nil)
	sibling := namedInst("Sibling", parent)
	child := namedInst("Child", inst)
	desc := namedInst("Descendant", child)

	if inst.Parent() != nil {
		t.Error("expected nil parent")
	}
	if err := inst.AddChildAt(100, inst); err == nil {
		t.Error("no error on adding self")
	}
	if err := child.AddChildAt(100, inst); err == nil {
		t.Error("no error on adding parent to child")
	}
	if err := desc.AddChildAt(100, inst); err == nil {
		t.Error("no error on adding parent to descendant")
	}
	if err := parent.AddChildAt(100, inst); err != nil {
		t.Error("failed add child:", err)
	}
	if inst.Parent() != parent {
		t.Error("unexpected parent")
	}
	if n := len(parent.Children); n != 2 {
		t.Error("unexpected length of children (expected 2, got %d):", n)
	}
	if parent.Children[0] != sibling {
		t.Error("unexpected sibling")
	}
	if parent.Children[1] != inst {
		t.Error("unexpected order of child")
	}
	if err := parent.AddChildAt(100, inst); err != nil {
		t.Error("error on adding same child:", err)
	}
	parent.AddChildAt(100, sibling)
	if parent.Children[0] != inst {
		t.Error("unexpected order of child")
	}
	if parent.Children[1] != sibling {
		t.Error("unexpected order of child")
	}

	parent = NewInstance("Parent", nil)
	child0 := NewInstance("Child0", nil)
	child1 := NewInstance("Child1", nil)
	child2 := NewInstance("Child2", nil)
	assertOrder := func(children ...*Instance) {
		if i, j := len(children), len(parent.Children); i != j {
			t.Error("unexpected number of children (expected %d, got %d)", i, j)
		}
		for i := 0; i < len(children); i++ {
			if parent.Children[i] != children[i] {
				t.Error("unexpected child %d (expected %s, got %s)", children[i].ClassName, parent.Children[i].ClassName)
			}
		}
	}
	clear := func() {
		child0.SetParent(nil)
		child1.SetParent(nil)
		child2.SetParent(nil)
	}

	assertOrder()

	parent.AddChildAt(0, child0)
	assertOrder(child0)
	parent.AddChildAt(1, child1)
	assertOrder(child0, child1)
	parent.AddChildAt(2, child2)
	assertOrder(child0, child1, child2)
	clear()

	parent.AddChildAt(-1, child0)
	assertOrder(child0)
	parent.AddChildAt(0, child1)
	assertOrder(child1, child0)
	parent.AddChildAt(1, child2)
	assertOrder(child1, child2, child0)
	parent.AddChildAt(0, child0)
	assertOrder(child0, child1, child2)
	parent.AddChildAt(1, child0)
	assertOrder(child1, child0, child2)
}

func testRemoveOrder(t *testing.T, at bool, parent *Instance, child []*Instance, remove, children string) {
	for _, c := range child {
		c.SetParent(nil)
	}
	for _, c := range child {
		c.SetParent(parent)
	}
	if at {
		for _, r := range remove {
			parent.RemoveChildAt(int(r - '0'))
		}
	} else {
		for _, r := range remove {
			parent.RemoveChild(child[int(r-'0')])
		}
	}
	if i, j := len(children), len(parent.Children); i != j {
		t.Errorf("%s:%s: unexpected number of children (expected %d, got %d)", remove, children, i, j)
	}
	for i, r := range children {
		c := child[int(r-'0')]
		if parent.Children[i] != c {
			t.Errorf("%s:%s: unexpected child %d (expected %s, got %s)", remove, children, i, c.ClassName, parent.Children[i].ClassName)
		}
	}
	for i, ch := range child {
		if bytes.Contains([]byte(children), []byte{byte(i + '0')}) {
			if ch.Parent() != parent {
				t.Errorf("%s:%s: expected parent of %s", remove, children, ch)
			}
		} else {
			if ch.Parent() != nil {
				t.Errorf("%s:%s: expected nil parent of %s", remove, children, ch)
			}
		}
	}
}

func TestInstance_RemoveChild(t *testing.T) {
	parent := NewInstance("Parent", nil)
	child := make([]*Instance, 3)
	for i := range child {
		child[i] = NewInstance("Child"+strconv.Itoa(i), nil)
	}
	// Remove child n | Verify child n
	testRemoveOrder(t, false, parent, child, "", "012")
	testRemoveOrder(t, false, parent, child, "0", "12")
	testRemoveOrder(t, false, parent, child, "1", "02")
	testRemoveOrder(t, false, parent, child, "2", "01")
	testRemoveOrder(t, false, parent, child, "00", "12")
	testRemoveOrder(t, false, parent, child, "10", "2")
	testRemoveOrder(t, false, parent, child, "20", "1")
	testRemoveOrder(t, false, parent, child, "01", "2")
	testRemoveOrder(t, false, parent, child, "11", "02")
	testRemoveOrder(t, false, parent, child, "21", "0")
	testRemoveOrder(t, false, parent, child, "02", "1")
	testRemoveOrder(t, false, parent, child, "12", "0")
	testRemoveOrder(t, false, parent, child, "22", "01")
	testRemoveOrder(t, false, parent, child, "000", "12")
	testRemoveOrder(t, false, parent, child, "100", "2")
	testRemoveOrder(t, false, parent, child, "200", "1")
	testRemoveOrder(t, false, parent, child, "010", "2")
	testRemoveOrder(t, false, parent, child, "110", "2")
	testRemoveOrder(t, false, parent, child, "210", "")
	testRemoveOrder(t, false, parent, child, "020", "1")
	testRemoveOrder(t, false, parent, child, "120", "")
	testRemoveOrder(t, false, parent, child, "220", "1")
	testRemoveOrder(t, false, parent, child, "001", "2")
	testRemoveOrder(t, false, parent, child, "101", "2")
	testRemoveOrder(t, false, parent, child, "201", "")
	testRemoveOrder(t, false, parent, child, "011", "2")
	testRemoveOrder(t, false, parent, child, "111", "02")
	testRemoveOrder(t, false, parent, child, "211", "0")
	testRemoveOrder(t, false, parent, child, "021", "")
	testRemoveOrder(t, false, parent, child, "121", "0")
	testRemoveOrder(t, false, parent, child, "221", "0")
	testRemoveOrder(t, false, parent, child, "002", "1")
	testRemoveOrder(t, false, parent, child, "102", "")
	testRemoveOrder(t, false, parent, child, "202", "1")
	testRemoveOrder(t, false, parent, child, "012", "")
	testRemoveOrder(t, false, parent, child, "112", "0")
	testRemoveOrder(t, false, parent, child, "212", "0")
	testRemoveOrder(t, false, parent, child, "022", "1")
	testRemoveOrder(t, false, parent, child, "122", "0")
	testRemoveOrder(t, false, parent, child, "222", "01")
}

func TestInstance_RemoveChildAt(t *testing.T) {
	parent := NewInstance("Parent", nil)
	child := make([]*Instance, 3)
	for i := range child {
		child[i] = NewInstance("Child"+strconv.Itoa(i), nil)
	}
	// Remove child at n | Verify child n
	testRemoveOrder(t, true, parent, child, "", "012")
	testRemoveOrder(t, true, parent, child, "0", "12")
	testRemoveOrder(t, true, parent, child, "1", "02")
	testRemoveOrder(t, true, parent, child, "2", "01")
	testRemoveOrder(t, true, parent, child, "00", "2")
	testRemoveOrder(t, true, parent, child, "10", "2")
	testRemoveOrder(t, true, parent, child, "20", "1")
	testRemoveOrder(t, true, parent, child, "01", "1")
	testRemoveOrder(t, true, parent, child, "11", "0")
	testRemoveOrder(t, true, parent, child, "21", "0")
	testRemoveOrder(t, true, parent, child, "02", "12")
	testRemoveOrder(t, true, parent, child, "12", "02")
	testRemoveOrder(t, true, parent, child, "22", "01")
	testRemoveOrder(t, true, parent, child, "000", "")
	testRemoveOrder(t, true, parent, child, "100", "")
	testRemoveOrder(t, true, parent, child, "200", "")
	testRemoveOrder(t, true, parent, child, "010", "")
	testRemoveOrder(t, true, parent, child, "110", "")
	testRemoveOrder(t, true, parent, child, "210", "")
	testRemoveOrder(t, true, parent, child, "020", "2")
	testRemoveOrder(t, true, parent, child, "120", "2")
	testRemoveOrder(t, true, parent, child, "220", "1")
	testRemoveOrder(t, true, parent, child, "001", "2")
	testRemoveOrder(t, true, parent, child, "101", "2")
	testRemoveOrder(t, true, parent, child, "201", "1")
	testRemoveOrder(t, true, parent, child, "011", "1")
	testRemoveOrder(t, true, parent, child, "111", "0")
	testRemoveOrder(t, true, parent, child, "211", "0")
	testRemoveOrder(t, true, parent, child, "021", "1")
	testRemoveOrder(t, true, parent, child, "121", "0")
	testRemoveOrder(t, true, parent, child, "221", "0")
	testRemoveOrder(t, true, parent, child, "002", "2")
	testRemoveOrder(t, true, parent, child, "102", "2")
	testRemoveOrder(t, true, parent, child, "202", "1")
	testRemoveOrder(t, true, parent, child, "012", "1")
	testRemoveOrder(t, true, parent, child, "112", "0")
	testRemoveOrder(t, true, parent, child, "212", "0")
	testRemoveOrder(t, true, parent, child, "022", "12")
	testRemoveOrder(t, true, parent, child, "122", "02")
	testRemoveOrder(t, true, parent, child, "222", "01")
}

func TestInstance_RemoveAll(t *testing.T) {
	parent := NewInstance("Parent", nil)
	children := make([]*Instance, 100)
	for i := range children {
		children[i] = NewInstance("Child", parent)
	}

	parent.RemoveAll()
	if i := len(parent.Children); i != 0 {
		t.Error("expected Children length of 0 (got %d)", i)
	}
	for i, child := range children {
		if child.Parent() != nil {
			t.Error("expected nil parent on child %d", i)
		}
	}
}

func TestInstance_Clone(t *testing.T) {
	inst := NewInstance("Instance", nil)
	inst.SetName("InstanceName")
	inst.Properties["Position"] = ValueVector3{X: 1, Y: 2, Z: 3}

	child := NewInstance("Child", inst)
	child.SetName("ChildName")
	child.Properties["Size"] = ValueVector3{X: 4, Y: 5, Z: 6}

	outside := NewInstance("Outside", nil)
	inst.Set("Reference", ValueReference{Instance: outside})

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
	a, b := inst.Get("Name").(ValueString), cinst.Get("Name").(ValueString)
	if a[0] = a[0] + 1; b[0] == a[0] {
		t.Error("slice of cloned Name poonts to same array as original")
	}
	if cinst.Properties["Position"] != inst.Properties["Position"] {
		t.Error("cloned Position property does not equal original")
	}
	if v, _ := cinst.Properties["Reference"].(ValueReference); v.Instance != outside {
		t.Error("cloned Reference property does not equal original")
	}

	var cchild *Instance
	if len(cinst.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(cinst.Children))
	} else {
		cchild = cinst.Children[0]
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

func TestInstance_Name(t *testing.T) {
	inst := NewInstance("Instance", nil)

	if inst.Name() != "" {
		t.Error("unexpected value returned from Name")
	}

	inst.SetName("Instance")

	if v, ok := inst.Properties["Name"]; !ok {
		t.Error("failed to set Name property")
	} else if v, ok := v.(ValueString); !ok {
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
	} else if v, ok := v.(ValueString); !ok {
		t.Error("expected ValueString type for Name property")
	} else if string(v) != "" {
		t.Error("unexpected value of Name property")
	}

	if inst.Name() != "" {
		t.Error("unexpected value returned from Name")
	}
}

func TestInstance_String(t *testing.T) {
	inst := NewInstance("Instance", nil)

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
	inst := NewInstance("Instance", nil)

	if inst.Get("Property") != nil {
		t.Error("unexpected value returned from Get")
	}

	inst.Set("Property", ValueString("Value"))

	if v, ok := inst.Properties["Property"]; !ok {
		t.Error("expected property")
	} else if v, ok := v.(ValueString); !ok {
		t.Error("expected ValueString type for property")
	} else if string(v) != "Value" {
		t.Error("unexpected value of property")
	}

	if v := inst.Get("Property"); v == nil {
		t.Error("expected property")
	} else if v, ok := v.(ValueString); !ok {
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
