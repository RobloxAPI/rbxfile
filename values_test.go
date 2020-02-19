// +build ignore

package rbxfile

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

var testTypes = []Type{}

func init() {
	testTypes = make([]Type, len(typeStrings))
	for i := range testTypes {
		testTypes[i] = Type(i + 1)
	}
}

func TestType_String(t *testing.T) {
	if TypeString.String() != "String" {
		t.Error("unexpected result")
	}

	if Type(0).String() != "Invalid" || Type(len(testTypes)+1).String() != "Invalid" {
		t.Error("expected Invalid string")
	}
}

func TestTypeFromString(t *testing.T) {
	for _, typ := range testTypes {
		if st := TypeFromString(typ.String()); st != typ {
			t.Errorf("expected type %s from TypeFromString (got %s)", typ, st)
		}
	}

	if TypeFromString("String") != TypeString {
		t.Error("unexpected result from TypeFromString")
	}

	if TypeFromString("UnknownType") != TypeInvalid {
		t.Error("unexpected result from TypeFromString")
	}
}

func TestNewValue(t *testing.T) {
	for _, typ := range testTypes {
		name := reflect.ValueOf(NewValue(typ)).Type().Name()
		if strings.TrimPrefix(name, "Value") != typ.String() {
			t.Errorf("type %s does not match Type%s", name, typ)
		}
	}
	if NewValue(TypeInvalid) != nil {
		t.Error("expected nil value for invalid type")
	}
}

func TestValueCopy(t *testing.T) {
	for _, typ := range testTypes {
		v := NewValue(typ)
		c := v.Copy()
		if !reflect.DeepEqual(v, c) {
			t.Errorf("copy of value %q is not equal to original", v.Type().String())
		}
	}
}

type testCompareString struct {
	v Value
	s string
}

func testCompareStrings(t *testing.T, vts []testCompareString) {
	for _, vt := range vts {
		if vt.v.String() != vt.s {
			t.Errorf("unexpected result from String method of value %q (%q expected, got %q)", vt.v.Type().String(), vt.s, vt.v.String())
		}
	}
}

func TestValueString(t *testing.T) {
	testCompareStrings(t, []testCompareString{
		{ValueString("test\000string"), "test\000string"},
		{ValueBinaryString("test\000string"), "test\000string"},
		{ValueProtectedString("test\000string"), "test\000string"},
		{ValueContent("test\000string"), "test\000string"},

		{ValueBool(true), "true"},
		{ValueBool(false), "false"},

		{ValueInt(42), "42"},
		{ValueInt(-42), "-42"},

		{ValueFloat(8388607.314159), "8388607.5"},
		{ValueFloat(math.Pi), "3.1415927"},
		{ValueFloat(-math.Phi), "-1.618034"},
		{ValueFloat(math.Inf(1)), "+Inf"},
		{ValueFloat(math.Inf(-1)), "-Inf"},
		{ValueFloat(math.NaN()), "NaN"},

		{ValueDouble(8388607.314159), "8388607.314159"},
		{ValueDouble(math.Pi), "3.141592653589793"},
		{ValueDouble(-math.Phi), "-1.618033988749895"},
		{ValueDouble(math.Inf(1)), "+Inf"},
		{ValueDouble(math.Inf(-1)), "-Inf"},
		{ValueDouble(math.NaN()), "NaN"},

		{ValueUDim{
			Scale:  math.Pi,
			Offset: 12345,
		}, "3.1415927, 12345"},

		{ValueUDim2{
			X: ValueUDim{
				Scale:  1,
				Offset: 2,
			},
			Y: ValueUDim{
				Scale:  3,
				Offset: 4,
			},
		}, "{1, 2}, {3, 4}"},

		{ValueRay{
			Origin:    ValueVector3{X: 1, Y: 2, Z: 3},
			Direction: ValueVector3{X: 4, Y: 5, Z: 6},
		}, "{1, 2, 3}, {4, 5, 6}"},

		{ValueFaces{
			Front:  true,
			Bottom: true,
			Left:   true,
			Back:   true,
			Top:    true,
			Right:  true,
		}, "Front, Bottom, Left, Back, Top, Right"},
		{ValueFaces{
			Front:  true,
			Bottom: false,
			Left:   true,
			Back:   false,
			Top:    true,
			Right:  false,
		}, "Front, Left, Top"},

		{ValueAxes{X: true, Y: true, Z: true}, "X, Y, Z"},
		{ValueAxes{X: true, Y: false, Z: true}, "X, Z"},

		{ValueBrickColor(194), "194"},

		{ValueColor3{R: 0.5, G: 0.25, B: 0.75}, "0.5, 0.25, 0.75"},

		{ValueVector2{X: 1, Y: 2}, "1, 2"},

		{ValueVector3{X: 1, Y: 2, Z: 3}, "1, 2, 3"},

		{ValueCFrame{
			Position: ValueVector3{X: 1, Y: 2, Z: 3},
			Rotation: [9]float32{4, 5, 6, 7, 8, 9, 10, 11, 12},
		}, "1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12"},

		{ValueToken(42), "42"},

		{ValueReference{}, "<nil>"},
		{ValueReference{Instance: namedInst("Instance", nil)}, "Instance"},

		{ValueVector3int16{X: 1, Y: 2, Z: 3}, "1, 2, 3"},

		{ValueVector2int16{X: 1, Y: 2}, "1, 2"},
	},
	)
}
