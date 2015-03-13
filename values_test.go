package rbxfile_test

import (
	"github.com/robloxapi/rbxfile"
	"math"
	"reflect"
	"testing"
)

func TestType_String(t *testing.T) {
	if rbxfile.TypeString.String() != "string" {
		t.Error("unexpected result from String")
	}

	if rbxfile.Type(0).String() != "Invalid" {
		t.Error("unexpected result from String")
	}
}

func TestTypeFromString(t *testing.T) {
	if rbxfile.TypeFromString("string") != rbxfile.TypeString {
		t.Error("unexpected result from TypeFromString")
	}

	if rbxfile.TypeFromString("UnknownType") != rbxfile.TypeInvalid {
		t.Error("unexpected result from TypeFromString")
	}
}

func TestNewValue(t *testing.T) {
	if _, ok := rbxfile.NewValue(rbxfile.TypeString).(rbxfile.ValueString); !ok {
		t.Error("expected ValueString from NewValue")
	}

	if rbxfile.NewValue(rbxfile.TypeInvalid) != nil {
		t.Error("expected nil value from NewValue")
	}
}

var types = []rbxfile.Type{
	rbxfile.TypeString,
	rbxfile.TypeBinaryString,
	rbxfile.TypeProtectedString,
	rbxfile.TypeContent,
	rbxfile.TypeBool,
	rbxfile.TypeInt,
	rbxfile.TypeFloat,
	rbxfile.TypeDouble,
	rbxfile.TypeUDim,
	rbxfile.TypeUDim2,
	rbxfile.TypeRay,
	rbxfile.TypeFaces,
	rbxfile.TypeAxes,
	rbxfile.TypeBrickColor,
	rbxfile.TypeColor3,
	rbxfile.TypeVector2,
	rbxfile.TypeVector3,
	rbxfile.TypeCFrame,
	rbxfile.TypeToken,
	rbxfile.TypeReference,
	rbxfile.TypeVector3int16,
	rbxfile.TypeVector2int16,
}

func TestValueType(t *testing.T) {
	for _, typ := range types {
		v := rbxfile.NewValue(typ)
		if v == nil || v.Type() != typ {
			t.Error("unexpected value from NewValue")
		}
	}
}

func TestValueCopy(t *testing.T) {
	for _, typ := range types {
		v := rbxfile.NewValue(typ)
		if !reflect.DeepEqual(v, v.Copy()) {
			t.Errorf("copy of value %q is not equal to original", v.Type().String())
		}
	}
}

type vtest struct {
	v rbxfile.Value
	s string
}

func compareStrings(t *testing.T, vts ...vtest) {
	for _, vt := range vts {
		if vt.v.String() != vt.s {
			t.Errorf("unexpected result from String method of value %q (%q expected, got %q)", vt.v.Type().String(), vt.s, vt.v.String())
		}
	}
}

func TestValueString(t *testing.T) {
	compareStrings(t,
		vtest{rbxfile.ValueString("test\000string"), "test\000string"},
		vtest{rbxfile.ValueBinaryString("test\000string"), "test\000string"},
		vtest{rbxfile.ValueProtectedString("test\000string"), "test\000string"},
		vtest{rbxfile.ValueContent("test\000string"), "test\000string"},

		vtest{rbxfile.ValueBool(true), "true"},
		vtest{rbxfile.ValueBool(false), "false"},

		vtest{rbxfile.ValueInt(42), "42"},
		vtest{rbxfile.ValueInt(-42), "-42"},

		vtest{rbxfile.ValueFloat(8388607.314159), "8388607.5"},
		vtest{rbxfile.ValueFloat(math.Pi), "3.1415927"},
		vtest{rbxfile.ValueFloat(-math.Phi), "-1.618034"},
		vtest{rbxfile.ValueFloat(math.Inf(1)), "+Inf"},
		vtest{rbxfile.ValueFloat(math.Inf(-1)), "-Inf"},
		vtest{rbxfile.ValueFloat(math.NaN()), "NaN"},

		vtest{rbxfile.ValueDouble(8388607.314159), "8388607.314159"},
		vtest{rbxfile.ValueDouble(math.Pi), "3.141592653589793"},
		vtest{rbxfile.ValueDouble(-math.Phi), "-1.618033988749895"},
		vtest{rbxfile.ValueDouble(math.Inf(1)), "+Inf"},
		vtest{rbxfile.ValueDouble(math.Inf(-1)), "-Inf"},
		vtest{rbxfile.ValueDouble(math.NaN()), "NaN"},

		vtest{rbxfile.ValueUDim{
			Scale:  math.Pi,
			Offset: 12345678,
		}, "3.1415927, 12345678"},

		vtest{rbxfile.ValueUDim2{
			X: rbxfile.ValueUDim{
				Scale:  1,
				Offset: 2,
			},
			Y: rbxfile.ValueUDim{
				Scale:  3,
				Offset: 4,
			},
		}, "{1, 2}, {3, 4}"},

		vtest{rbxfile.ValueRay{
			Origin:    rbxfile.ValueVector3{X: 1, Y: 2, Z: 3},
			Direction: rbxfile.ValueVector3{X: 4, Y: 5, Z: 6},
		}, "{1, 2, 3}, {4, 5, 6}"},

		vtest{rbxfile.ValueFaces{
			Front:  true,
			Bottom: true,
			Left:   true,
			Back:   true,
			Top:    true,
			Right:  true,
		}, "Front, Bottom, Left, Back, Top, Right"},
		vtest{rbxfile.ValueFaces{
			Front:  true,
			Bottom: false,
			Left:   true,
			Back:   false,
			Top:    true,
			Right:  false,
		}, "Front, Left, Top"},

		vtest{rbxfile.ValueAxes{X: true, Y: true, Z: true}, "X, Y, Z"},
		vtest{rbxfile.ValueAxes{X: true, Y: false, Z: true}, "X, Z"},

		vtest{rbxfile.ValueBrickColor(194), "194"},

		vtest{rbxfile.ValueColor3{R: 0.5, G: 0.25, B: 0.75}, "0.5, 0.25, 0.75"},

		vtest{rbxfile.ValueVector2{X: 1, Y: 2}, "1, 2"},

		vtest{rbxfile.ValueVector3{X: 1, Y: 2, Z: 3}, "1, 2, 3"},

		vtest{rbxfile.ValueCFrame{
			Position: rbxfile.ValueVector3{X: 1, Y: 2, Z: 3},
			Rotation: [9]float32{4, 5, 6, 7, 8, 9, 10, 11, 12},
		}, "1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12"},

		vtest{rbxfile.ValueToken(42), "42"},

		vtest{rbxfile.ValueReference{}, "<nil>"},
		vtest{rbxfile.ValueReference{Instance: namedInst("Instance", nil)}, "Instance"},

		vtest{rbxfile.ValueVector3int16{X: 1, Y: 2, Z: 3}, "1, 2, 3"},

		vtest{rbxfile.ValueVector2int16{X: 1, Y: 2}, "1, 2"},
	)
}
