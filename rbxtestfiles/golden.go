package main

import (
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/robloxapi/rbxfile"
	"github.com/robloxapi/rbxfile/bin"
	"github.com/robloxapi/rbxfile/xml"
)

type Golden struct {
	s         strings.Builder
	lead      []byte
	format    string
	structure string
	refs      map[*rbxfile.Instance]int
}

func (g *Golden) push() *Golden {
	g.lead = append(g.lead, '\t')
	return g
}

func (g *Golden) pop() *Golden {
	g.lead = g.lead[:len(g.lead)-1]
	return g
}

func (g *Golden) newline() *Golden {
	g.s.WriteByte('\n')
	g.s.Write(g.lead)
	return g
}

type array []interface{}

// array writes v as a JSON array.
func (g *Golden) array(v array) *Golden {
	g.s.WriteByte('[')
	if len(v) == 0 {
		g.s.WriteByte(']')
		return g
	}
	g.push()
	g.newline()
	g.value(v[0])
	for i := 1; i < len(v); i++ {
		g.s.WriteByte(',')
		g.newline()
		g.value(v[i])
	}
	g.pop()
	g.newline()
	g.s.WriteByte(']')
	return g
}

type object []field
type field struct {
	name  string
	value interface{}
}

// object writes v as a JSON object.
func (g *Golden) object(v object) *Golden {
	g.s.WriteByte('{')
	if len(v) == 0 {
		g.s.WriteByte('}')
		return g
	}
	g.push()
	g.newline()
	g.string(v[0].name)
	g.s.WriteString(": ")
	g.value(v[0].value)
	for i := 1; i < len(v); i++ {
		g.s.WriteByte(',')
		g.newline()
		g.string(v[i].name)
		g.s.WriteString(": ")
		g.value(v[i].value)
	}
	g.pop()
	g.newline()
	g.s.WriteByte('}')
	return g
}

func (g *Golden) string(s string) *Golden {
	// From encoding/json
	const hex = "0123456789abcdef"
	g.s.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if b >= ' ' && b != '"' && b != '\\' {
				i++
				continue
			}
			if start < i {
				g.s.WriteString(s[start:i])
			}
			g.s.WriteByte('\\')
			switch b {
			case '\\', '"':
				g.s.WriteByte(b)
			case '\n':
				g.s.WriteByte('n')
			case '\r':
				g.s.WriteByte('r')
			case '\t':
				g.s.WriteByte('t')
			default:
				g.s.WriteString(`u00`)
				g.s.WriteByte(hex[b>>4])
				g.s.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				g.s.WriteString(s[start:i])
			}
			g.s.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				g.s.WriteString(s[start:i])
			}
			g.s.WriteString(`\u202`)
			g.s.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		g.s.WriteString(s[start:])
	}
	g.s.WriteByte('"')
	return g
}

func (g *Golden) ref(v *rbxfile.Instance) *Golden {
	if i, ok := g.refs[v]; ok {
		return g.value(i)
	}
	return g.value(nil)
}

func recurseRefs(refs map[*rbxfile.Instance]int, instances []*rbxfile.Instance) {
	for _, inst := range instances {
		if _, ok := refs[inst]; !ok {
			refs[inst] = len(refs)
			recurseRefs(refs, inst.Children)
		}
	}
}

func (g *Golden) value(v interface{}) *Golden {
	switch v := v.(type) {
	default:
		g.s.WriteString("<UNKNOWN:" + reflect.TypeOf(v).String() + ">")

	case nil:
		g.s.WriteString("null")
	case bool:
		if v {
			g.s.WriteString("true")
		} else {
			g.s.WriteString("false")
		}
	case string:
		for _, r := range []rune(v) {
			switch r {
			case '\t', '\n':
				continue
			}
			if !unicode.IsGraphic(r) {
				return g.value([]byte(v))
			}
		}
		g.string(v)
	case uint:
		g.s.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint8:
		g.s.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint16:
		g.s.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint32:
		g.s.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint64:
		g.s.WriteString(strconv.FormatUint(v, 10))
	case int:
		g.s.WriteString(strconv.FormatInt(int64(v), 10))
	case int8:
		g.s.WriteString(strconv.FormatInt(int64(v), 10))
	case int16:
		g.s.WriteString(strconv.FormatInt(int64(v), 10))
	case int32:
		g.s.WriteString(strconv.FormatInt(int64(v), 10))
	case int64:
		g.s.WriteString(strconv.FormatInt(int64(v), 10))
	case float32:
		g.s.WriteString(strconv.FormatFloat(float64(v), 'g', 9, 32))
	case float64:
		g.s.WriteString(strconv.FormatFloat(v, 'g', 17, 64))

	case []byte:
		if len(v) == 0 {
			g.s.WriteString("[]")
			break
		}
		g.s.WriteByte('[')
		g.push()
		for i, c := range v {
			if i%16 == 0 {
				g.s.WriteByte('\n')
				g.s.Write(g.lead)
			}
			if c < 100 {
				g.s.WriteByte(' ')
				if c < 10 {
					g.s.WriteByte(' ')
				}
			}
			g.s.WriteString(strconv.FormatUint(uint64(c), 10))
			if i < len(v)-1 {
				g.s.WriteByte(',')
			}
		}
		g.pop()
		g.newline()
		g.s.WriteByte(']')

	case error:
		g.value(v.Error())

	case array:
		g.array(v)

	case object:
		g.object(v)

	case map[string]string:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		a := make(array, len(keys))
		for i, k := range keys {
			a[i] = object{
				{name: "Key", value: k},
				{name: "Value", value: v[k]},
			}
		}
		g.array(a)

	case *rbxfile.Root:
		// Prepopulate ref table.
		g.refs = map[*rbxfile.Instance]int{}
		recurseRefs(g.refs, v.Instances)

		g.object(object{
			{name: "Metadata", value: v.Metadata},
			{name: "Instances", value: v.Instances},
		})

	case map[string]rbxfile.Value:
		props := make([]string, 0, len(v))
		for name := range v {
			props = append(props, name)
		}
		sort.Strings(props)

		a := make(array, len(props))
		for i, name := range props {
			value := v[name]
			a[i] = object{
				{name: "Name", value: name},
				{name: "Type", value: value.Type().String()},
				{name: "Value", value: value},
			}
		}
		g.array(a)

	case []*rbxfile.Instance:
		a := make(array, len(v))
		for i, inst := range v {
			a[i] = inst
		}
		g.array(a)

	case *rbxfile.Instance:
		var ref interface{}
		if r, ok := g.refs[v]; ok {
			ref = r
		}
		g.object(object{
			field{name: "ClassName", value: v.ClassName},
			field{name: "IsService", value: v.IsService},
			field{name: "Reference", value: ref},
			field{name: "Properties", value: v.Properties},
			field{name: "Children", value: v.Children},
		})

	case rbxfile.ValueString:
		g.value(string(v))

	case rbxfile.ValueBinaryString:
		g.value([]byte(v))

	case rbxfile.ValueProtectedString:
		g.value(string(v))

	case rbxfile.ValueContent:
		g.value(string(v))

	case rbxfile.ValueBool:
		g.value(bool(v))

	case rbxfile.ValueInt:
		g.value(int64(v))

	case rbxfile.ValueFloat:
		g.value(float32(v))

	case rbxfile.ValueDouble:
		g.value(float64(v))

	case rbxfile.ValueUDim:
		g.object(object{
			field{name: "Scale", value: v.Scale},
			field{name: "Offset", value: v.Offset},
		})

	case rbxfile.ValueUDim2:
		g.object(object{
			field{name: "X", value: v.X},
			field{name: "Y", value: v.Y},
		})

	case rbxfile.ValueRay:
		g.object(object{
			field{name: "Origin", value: v.Origin},
			field{name: "Direction", value: v.Direction},
		})

	case rbxfile.ValueFaces:
		g.object(object{
			field{name: "Right", value: v.Right},
			field{name: "Top", value: v.Top},
			field{name: "Back", value: v.Back},
			field{name: "Left", value: v.Left},
			field{name: "Bottom", value: v.Bottom},
			field{name: "Front", value: v.Front},
		})

	case rbxfile.ValueAxes:
		g.object(object{
			field{name: "X", value: v.X},
			field{name: "Y", value: v.Y},
			field{name: "Z", value: v.Z},
		})

	case rbxfile.ValueBrickColor:
		g.value(uint32(v))

	case rbxfile.ValueColor3:
		g.object(object{
			field{name: "R", value: v.R},
			field{name: "G", value: v.G},
			field{name: "B", value: v.B},
		})

	case rbxfile.ValueVector2:
		g.object(object{
			field{name: "X", value: v.X},
			field{name: "Y", value: v.Y},
		})

	case rbxfile.ValueVector3:
		g.object(object{
			field{name: "X", value: v.X},
			field{name: "Y", value: v.Y},
			field{name: "Z", value: v.Z},
		})

	case rbxfile.ValueCFrame:
		g.object(object{
			field{name: "Position", value: v.Position},
			field{name: "Rotation", value: object{
				field{name: "R00", value: v.Rotation[0]},
				field{name: "R01", value: v.Rotation[1]},
				field{name: "R02", value: v.Rotation[2]},
				field{name: "R10", value: v.Rotation[3]},
				field{name: "R11", value: v.Rotation[4]},
				field{name: "R12", value: v.Rotation[5]},
				field{name: "R20", value: v.Rotation[6]},
				field{name: "R21", value: v.Rotation[7]},
				field{name: "R22", value: v.Rotation[8]},
			}},
		})

	case rbxfile.ValueToken:
		g.value(uint32(v))

	case rbxfile.ValueReference:
		g.ref(v.Instance)

	case rbxfile.ValueVector3int16:
		g.object(object{
			field{name: "X", value: v.X},
			field{name: "Y", value: v.Y},
			field{name: "Z", value: v.Z},
		})

	case rbxfile.ValueVector2int16:
		g.object(object{
			field{name: "X", value: v.X},
			field{name: "Y", value: v.Y},
		})

	case rbxfile.ValueNumberSequenceKeypoint:
		g.object(object{
			field{name: "Time", value: v.Time},
			field{name: "Value", value: v.Value},
			field{name: "Envelope", value: v.Envelope},
		})

	case rbxfile.ValueNumberSequence:
		a := make(array, len(v))
		for i, k := range v {
			a[i] = k
		}
		g.array(a)

	case rbxfile.ValueColorSequenceKeypoint:
		g.object(object{
			field{name: "Time", value: v.Time},
			field{name: "Value", value: v.Value},
			field{name: "Envelope", value: v.Envelope},
		})

	case rbxfile.ValueColorSequence:
		a := make(array, len(v))
		for i, k := range v {
			a[i] = k
		}
		g.array(a)

	case rbxfile.ValueNumberRange:
		g.object(object{
			field{name: "Min", value: v.Min},
			field{name: "Max", value: v.Max},
		})

	case rbxfile.ValueRect2D:
		g.object(object{
			field{name: "Min", value: v.Min},
			field{name: "Max", value: v.Max},
		})

	case rbxfile.ValuePhysicalProperties:
		if v.CustomPhysics {
			g.object(object{
				field{name: "CustomPhysics", value: v.CustomPhysics},
				field{name: "Density", value: v.Density},
				field{name: "Friction", value: v.Friction},
				field{name: "Elasticity", value: v.Elasticity},
				field{name: "FrictionWeight", value: v.FrictionWeight},
				field{name: "ElasticityWeight", value: v.ElasticityWeight},
			})
		} else {
			g.object(object{
				field{name: "CustomPhysics", value: v.CustomPhysics},
			})
		}

	case rbxfile.ValueColor3uint8:
		g.object(object{
			field{name: "R", value: v.R},
			field{name: "G", value: v.G},
			field{name: "B", value: v.B},
		})

	case rbxfile.ValueInt64:
		g.value(int64(v))

	case rbxfile.ValueSharedString:
		g.value(string(v))

	case *bin.FormatModel:
		chunks := make(array, len(v.Chunks))
		for i, chunk := range v.Chunks {
			chunks[i] = chunk
		}
		g.object(object{
			field{name: "Version", value: v.Version},
			field{name: "Types", value: v.TypeCount},
			field{name: "Instances", value: v.InstanceCount},
			field{name: "Chunks", value: chunks},
		})

	case *bin.ChunkMeta:
		sig := v.Signature()
		values := make(array, len(v.Values))
		for i, s := range v.Values {
			values[i] = object{
				field{name: "Key", value: s[0]},
				field{name: "Value", value: s[1]},
			}
		}
		g.object(object{
			field{name: "Signature", value: sig[:]},
			field{name: "Compressed", value: v.IsCompressed},
			field{name: "Values", value: values},
		})

	case *bin.ChunkSharedStrings:
		sig := v.Signature()
		values := make(array, len(v.Values))
		for i, s := range v.Values {
			values[i] = object{
				field{name: "Hash", value: s.Hash[:]},
				field{name: "Value", value: s.Value},
			}
		}
		g.object(object{
			field{name: "Signature", value: sig[:]},
			field{name: "Compressed", value: v.IsCompressed},
			field{name: "Version", value: v.Version},
			field{name: "Values", value: values},
		})

	case *bin.ChunkInstance:
		sig := v.Signature()

		instanceIDs := make(array, len(v.InstanceIDs))
		for i, id := range v.InstanceIDs {
			instanceIDs[i] = id
		}
		getService := make(array, len(v.GetService))
		for i, s := range v.GetService {
			getService[i] = s
		}
		g.object(object{
			field{name: "Signature", value: sig[:]},
			field{name: "Compressed", value: v.IsCompressed},
			field{name: "TypeID", value: v.TypeID},
			field{name: "ClassName", value: v.ClassName},
			field{name: "InstanceIDs", value: instanceIDs},
			field{name: "IsService", value: v.IsService},
			field{name: "GetService", value: getService},
		})

	case *bin.ChunkProperty:
		sig := v.Signature()
		props := make(array, len(v.Properties))
		for i, prop := range v.Properties {
			props[i] = prop
		}
		g.object(object{
			field{name: "Signature", value: sig[:]},
			field{name: "Compressed", value: v.IsCompressed},
			field{name: "TypeID", value: v.TypeID},
			field{name: "PropertyName", value: v.PropertyName},
			field{name: "DataType", value: "0x" + strconv.FormatUint(uint64(v.DataType), 16) + " (" + v.DataType.String() + ")"},
			field{name: "Values", value: props},
		})

	case *bin.ChunkParent:
		sig := v.Signature()
		children := make(array, len(v.Children))
		for i, child := range v.Children {
			children[i] = child
		}
		parents := make(array, len(v.Parents))
		for i, parent := range v.Parents {
			parents[i] = parent
		}
		g.object(object{
			field{name: "Signature", value: sig[:]},
			field{name: "Compressed", value: v.IsCompressed},
			field{name: "Version", value: v.Version},
			field{name: "Children", value: children},
			field{name: "Parents", value: parents},
		})

	case *bin.ChunkEnd:
		sig := v.Signature()
		g.object(object{
			field{name: "Signature", value: sig[:]},
			field{name: "Compressed", value: v.IsCompressed},
			field{name: "Content", value: v.Content},
		})
	}
	return g
}

func (g *Golden) Format(format string, v interface{}) *Golden {
	g.format = format
	switch v.(type) {
	case error:
		g.structure = "error"
	case *rbxfile.Root:
		g.structure = "model"
	case *bin.FormatModel:
		g.structure = "binary"
	case *xml.Document:
		g.structure = "xml"
	}
	g.object(object{
		field{name: "Format", value: g.format},
		field{name: "Output", value: g.structure},
		field{name: "Data", value: v},
	})
	return g
}

func (g *Golden) Bytes() []byte {
	return []byte(g.s.String())
}
