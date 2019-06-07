// The declare package is used to generate rbxfile structures in a declarative
// style.
//
// Most items have a Declare method, which returns a new rbxfile structure
// corresponding to the declared item.
//
// The easiest way to use this package is to import it directly into the
// current package:
//
//     import . "github.com/robloxapi/rbxfile/declare"
//
// This allows the package's identifiers to be used directly without a
// qualifier.
package declare

import (
	"github.com/robloxapi/rbxfile"
)

// primary is implemented by declarations that can be directly within a Root
// declaration.
type primary interface {
	primary()
}

// Root declares a rbxfile.Root. It is a list that contains Instance and
// Metadata declarations.
type Root []primary

// build recursively resolves instance declarations.
func build(dinst instance, refs rbxfile.References, props map[*rbxfile.Instance][]property) *rbxfile.Instance {
	inst := rbxfile.NewInstance(dinst.className, nil)

	if dinst.reference != "" {
		refs[dinst.reference] = inst
		inst.Reference = dinst.reference
	}

	inst.Properties = make(map[string]rbxfile.Value, len(dinst.properties))
	props[inst] = dinst.properties

	for _, dchild := range dinst.children {
		child := build(dchild, refs, props)
		inst.AddChild(child)
	}

	return inst
}

// Declare evaluates the Root declaration, generating instances, metadata, and
// property values, setting up the instance hierarchy, and resolving references.
//
// Elements are evaluated in order; if two metadata declarations have the same
// key, the latter takes precedence.
func (droot Root) Declare() *rbxfile.Root {
	lenInst := 0
	lenMeta := 0
	for _, p := range droot {
		switch p.(type) {
		case instance:
			lenInst++
		case metadata:
			lenMeta++
		}
	}

	root := rbxfile.Root{
		Instances: make([]*rbxfile.Instance, 0, lenInst),
		Metadata:  make(map[string]string, lenMeta),
	}

	refs := rbxfile.References{}
	props := map[*rbxfile.Instance][]property{}

	for _, p := range droot {
		switch p := p.(type) {
		case instance:
			root.Instances = append(root.Instances, build(p, refs, props))
		case metadata:
			root.Metadata[p[0]] = p[1]
		}
	}

	for inst, properties := range props {
		for _, prop := range properties {
			inst.Properties[prop.name] = prop.typ.value(refs, prop.value)
		}
	}

	return &root
}

// metadata represents the declaration of metadata.
type metadata [2]string

func (metadata) primary() {}

// Metadata declares key-value pair to be applied to the root's Metadata field.
func Metadata(key, value string) metadata {
	return metadata{key, value}
}

// element is implemented by declarations that can be within an instance
// declaration.
type element interface {
	element()
}

// instance represents the declaration of a rbxfile.Instance.
type instance struct {
	className  string
	reference  string
	properties []property
	children   []instance
}

func (instance) primary() {}
func (instance) element() {}

// Declare evaluates the Instance declaration, generating the instance,
// descendants, and property values, setting up the instance hierarchy, and
// resolving references.
func (dinst instance) Declare() *rbxfile.Instance {
	inst := rbxfile.NewInstance(dinst.className, nil)

	refs := rbxfile.References{}
	props := map[*rbxfile.Instance][]property{}

	if dinst.reference != "" {
		refs[dinst.reference] = inst
		inst.Reference = dinst.reference
	}

	inst.Properties = make(map[string]rbxfile.Value, len(dinst.properties))
	props[inst] = dinst.properties

	for _, dchild := range dinst.children {
		child := build(dchild, refs, props)
		inst.AddChild(child)
	}

	for inst, properties := range props {
		for _, prop := range properties {
			inst.Properties[prop.name] = prop.typ.value(refs, prop.value)
		}
	}

	return inst
}

// Instance declares a rbxfile.Instance. It defines an instance with a class
// name, and a series of "elements". An element can be a Property declaration,
// which defines a property for the instance. An element can also be another
// Instance declaration, which becomes a child of the instance.
//
// An element can also be a "Ref" declaration, which defines a string that can
// be used to refer to the instance by properties with the Reference value
// type. This also sets the instance's Reference field.
func Instance(className string, elements ...element) instance {
	inst := instance{
		className: className,
	}

	for _, e := range elements {
		switch e := e.(type) {
		case Ref:
			inst.reference = string(e)
		case property:
			inst.properties = append(inst.properties, e)
		case instance:
			inst.children = append(inst.children, e)
		}
	}

	return inst
}

type property struct {
	name  string
	typ   Type
	value []interface{}
}

func (property) element() {}

// Property declares a property of a rbxfile.Instance. It defines the name of
// the property, a type corresponding to a rbxfile.Value, and the value of the
// property.
//
// The value argument may be one or more values of any type, which are
// asserted to a rbxfile.Value corresponding to the given type. If the
// value(s) cannot be asserted, then the zero value for the given type is
// returned instead.
//
// When the given type or a field of the given type is a number, any number
// type except for complex numbers may be given as the value.
//
// The value may be a single rbxfile.Value that corresponds to the given type
// (e.g. rbxfile.ValueString for String), in which case the value itself is
// returned.
//
// Otherwise, for a given type, values must be the following:
//
//     String, BinaryString, ProtectedString, Content:
//         A single string or []byte. Extra values are ignored.
//
//     Bool:
//         A single bool. Extra values are ignored.
//
//     Int, Float, Double, BrickColor, Token, Int64:
//         A single number. Extra values are ignored.
//
//     UDim:
//         2 numbers, corresponding to the Scale and Offset fields.
//
//     UDim2:
//         1) 2 rbxfile.ValueUDims, corresponding to the X and Y fields.
//         2) 4 numbers, corresponding to the X.Scale, X.Offset, Y.Scale, and
//            Y.Offset fields.
//
//     Ray:
//         1) 2 rbxfile.ValueVector3s, corresponding to the Origin and
//            Direction fields.
//         2) 6 numbers, corresponding to the X, Y, and Z fields of Origin,
//            then of Direction.
//
//     Faces:
//         6 bools, corresponding to the Right, Top, Back, Left, Bottom, and
//         Front fields.
//
//     Axes:
//         3 bools, corresponding to the X, Y, and Z fields.
//
//     Color3:
//         3 numbers, corresponding to the R, G, and B fields.
//
//     Vector2, Vector2int16:
//         2 numbers, corresponding to the X and Y fields.
//
//     Vector3, Vector3int16:
//         3 numbers, corresponding to the X, Y, and Z fields.
//
//     CFrame:
//         1) 10 values. The first value must be a rbxfile.ValueVector3, which
//            corresponds to the Position field. The remaining 9 values must
//            be numbers, which correspond to the components of the Rotation
//            field.
//         2) 12 numbers. The first 3 correspond to the X, Y, and Z fields of
//            the Position field. The remaining 9 numbers correspond to the
//            Rotation field.
//
//     Reference:
//         A single string, []byte or *rbxfile.Instance. Extra values are
//         ignored. When the value is a string or []byte, the reference is
//         resolved by looking for an instance whose "Ref" declaration is
//         equal to the value.
//
//     NumberSequence:
//         1) 2 or more rbxfile.ValueNumberSequenceKeypoints, which correspond
//            to keypoints in the sequence.
//         2) 2 or more groups of 3 numbers. Each group corresponds to the
//            fields Time, Value, and Envelope of a single keypoint in the
//            sequence.
//
//     ColorSequence:
//         1) 2 or more rbxfile.ValueColorSequenceKeypoints, which correspond
//            to keypoints in the sequence.
//         2) 2 or more groups of 3 values: A number, a rbxfile.ValueColor3,
//            and a number. Each group corresponds to the Time, Value and
//            Envelope fields of a single keypoint in the sequence.
//         3) 2 or more groups of 5 numbers. Each group corresponds to the
//            fields Time, Value.R, Value.G, Value.B, and Envelope of a single
//            keypoint in the sequence.
//
//     NumberRange:
//         2 numbers, corresponding to the Min and Max fields.
//
//     Rect2D:
//         1) 2 rbxfile.ValueVector2s, corresponding to the Min and Max
//            fields.
//         2) 4 numbers, corresponding to the Min.X, Min.Y, Max.X, and Max.Y
//            fields.
//
//     PhysicalProperties:
//         1) No values, indicating PhysicalProperties with CustomPhysics set
//            to false.
//         2) 3 numbers, corresponding to the Density, Friction, and
//            Elasticity fields (CustomPhysics is set to true).
//         3) 5 numbers, corresponding to the Density, Friction, and
//            Elasticity, FrictionWeight, and ElasticityWeight fields
//            (CustomPhysics is set to true).
//
//     Color3uint8:
//         3 numbers, corresponding to the R, G, and B fields.
func Property(name string, typ Type, value ...interface{}) property {
	return property{name: name, typ: typ, value: value}
}

// Declare evaluates the Property declaration. Since the property does not
// belong to any instance, the name is ignored, and only the value is
// generated.
func (prop property) Declare() rbxfile.Value {
	var refs rbxfile.References
	return prop.typ.value(refs, prop.value)
}

// Ref declares a string that can be used to refer to the Instance under which
// it was declared. This will also set the instance's Reference field.
type Ref string

func (Ref) element() {}
