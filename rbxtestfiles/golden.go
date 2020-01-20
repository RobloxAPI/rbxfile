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

func (g *Golden) pushObject() *Golden {
	g.s.WriteByte('{')
	g.push()
	return g
}

func (g *Golden) pushObjectf(field string) *Golden {
	g.string(field)
	g.s.WriteString(": {")
	g.push()
	return g
}

func (g *Golden) popObject(sep bool) *Golden {
	g.pop()
	g.newline()
	g.s.WriteByte('}')
	if sep {
		g.s.WriteByte(',')
	}
	return g
}

func (g *Golden) pushArray() *Golden {
	g.s.WriteByte('[')
	g.push()
	return g
}

func (g *Golden) pushArrayf(field string) *Golden {
	g.string(field)
	g.s.WriteString(": [")
	g.push()
	return g
}

func (g *Golden) popArray(newline, sep bool) *Golden {
	g.pop()
	if newline {
		g.newline()
	}
	g.s.WriteByte(']')
	if sep {
		g.s.WriteByte(',')
	}
	return g
}

func (g *Golden) field(f string, v interface{}, sep bool) *Golden {
	g.newline()
	if f != "" {
		g.string(f)
		g.s.WriteString(": ")
	}
	g.value(v)
	if sep {
		g.s.WriteByte(',')
	}
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
		g.pushArray()
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
		g.popArray(len(v) > 0, false)

	case error:
		g.value(v.Error())

	case map[string]string:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		g.pushArray()
		for i, k := range keys {
			g.newline()
			g.pushObject()
			g.field("Key", k, true)
			g.field("Value", v[k], false)
			g.popObject(i < len(keys)-1)
		}
		g.popArray(len(v) > 0, false)

	case *rbxfile.Root:
		// Prepopulate ref table.
		g.refs = map[*rbxfile.Instance]int{}
		recurseRefs(g.refs, v.Instances)

		g.pushObject()
		g.field("Metadata", v.Metadata, true)
		g.field("Instances", v.Instances, false)
		g.popObject(false)

	case map[string]rbxfile.Value:
		props := make([]string, 0, len(v))
		for name := range v {
			props = append(props, name)
		}
		sort.Strings(props)
		g.pushArray()
		for i, name := range props {
			value := v[name]
			g.newline()
			g.pushObject()
			g.field("Name", name, true)
			g.field("Type", value.Type().String(), true)
			g.field("Value", value, false)
			g.popObject(i < len(props)-1)
		}
		g.popArray(len(props) > 0, false)

	case []*rbxfile.Instance:
		g.pushArray()
		for i, inst := range v {
			g.field("", inst, i < len(v)-1)
		}
		g.popArray(len(v) > 0, false)

	case *rbxfile.Instance:
		g.pushObject()
		g.field("ClassName", v.ClassName, true)
		g.field("IsService", v.IsService, true)
		if ref, ok := g.refs[v]; ok {
			g.field("Reference", ref, true)
		} else {
			g.field("Reference", nil, true)
		}
		g.field("Properties", v.Properties, true)
		g.field("Children", v.Children, false)
		g.popObject(false)

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
		g.pushObject()
		g.field("Scale", v.Scale, true)
		g.field("Offset", v.Offset, false)
		g.popObject(false)

	case rbxfile.ValueUDim2:
		g.pushObject()
		g.field("X", v.X, true)
		g.field("Y", v.Y, false)
		g.popObject(false)

	case rbxfile.ValueRay:
		g.pushObject()
		g.field("Origin", v.Origin, true)
		g.field("Direction", v.Direction, false)
		g.popObject(false)

	case rbxfile.ValueFaces:
		g.pushObject()
		g.field("Right", v.Right, true)
		g.field("Top", v.Top, true)
		g.field("Back", v.Back, true)
		g.field("Left", v.Left, true)
		g.field("Bottom", v.Bottom, true)
		g.field("Front", v.Front, false)
		g.popObject(false)

	case rbxfile.ValueAxes:
		g.pushObject()
		g.field("X", v.X, true)
		g.field("Y", v.Y, true)
		g.field("Z", v.Z, false)
		g.popObject(false)

	case rbxfile.ValueBrickColor:
		g.value(uint32(v))

	case rbxfile.ValueColor3:
		g.pushObject()
		g.field("R", v.R, true)
		g.field("G", v.G, true)
		g.field("B", v.B, false)
		g.popObject(false)

	case rbxfile.ValueVector2:
		g.pushObject()
		g.field("X", v.X, true)
		g.field("Y", v.Y, false)
		g.popObject(false)

	case rbxfile.ValueVector3:
		g.pushObject()
		g.field("X", v.X, true)
		g.field("Y", v.Y, true)
		g.field("Z", v.Z, false)
		g.popObject(false)

	case rbxfile.ValueCFrame:
		g.pushObject()
		g.field("Position", v.Position, true)
		g.pushObjectf("Rotation")
		g.field("R00", v.Rotation[0], true)
		g.field("R01", v.Rotation[1], true)
		g.field("R02", v.Rotation[2], true)
		g.field("R10", v.Rotation[3], true)
		g.field("R11", v.Rotation[4], true)
		g.field("R12", v.Rotation[5], true)
		g.field("R20", v.Rotation[6], true)
		g.field("R21", v.Rotation[7], true)
		g.field("R22", v.Rotation[8], false)
		g.popObject(false)
		g.popObject(false)

	case rbxfile.ValueToken:
		g.value(uint32(v))

	case rbxfile.ValueReference:
		g.ref(v.Instance)

	case rbxfile.ValueVector3int16:
		g.pushObject()
		g.field("X", v.X, true)
		g.field("Y", v.Y, true)
		g.field("Z", v.Z, false)
		g.popObject(false)

	case rbxfile.ValueVector2int16:
		g.pushObject()
		g.field("X", v.X, true)
		g.field("Y", v.Y, false)
		g.popObject(false)

	case rbxfile.ValueNumberSequenceKeypoint:
		g.pushObject()
		g.field("Time", v.Time, true)
		g.field("Value", v.Value, true)
		g.field("Envelope", v.Envelope, false)
		g.popObject(false)

	case rbxfile.ValueNumberSequence:
		g.pushArray()
		for i, k := range v {
			g.field("", k, i < len(v)-1)
		}
		g.popArray(len(v) > 0, false)

	case rbxfile.ValueColorSequenceKeypoint:
		g.pushObject()
		g.field("Time", v.Time, true)
		g.field("Value", v.Value, true)
		g.field("Envelope", v.Envelope, false)
		g.popObject(false)

	case rbxfile.ValueColorSequence:
		g.pushArray()
		for i, k := range v {
			g.field("", k, i < len(v)-1)
		}
		g.popArray(len(v) > 0, false)

	case rbxfile.ValueNumberRange:
		g.pushObject()
		g.field("Min", v.Min, true)
		g.field("Max", v.Max, false)
		g.popObject(false)

	case rbxfile.ValueRect2D:
		g.pushObject()
		g.field("Min", v.Min, true)
		g.field("Max", v.Max, false)
		g.popObject(false)

	case rbxfile.ValuePhysicalProperties:
		g.pushObject()
		g.field("CustomPhysics", v.CustomPhysics, v.CustomPhysics)
		if v.CustomPhysics {
			g.field("Density", v.Density, true)
			g.field("Friction", v.Friction, true)
			g.field("Elasticity", v.Elasticity, true)
			g.field("FrictionWeight", v.FrictionWeight, true)
			g.field("ElasticityWeight", v.ElasticityWeight, false)
		}
		g.popObject(false)

	case rbxfile.ValueColor3uint8:
		g.pushObject()
		g.field("R", v.R, true)
		g.field("G", v.G, true)
		g.field("B", v.B, false)
		g.popObject(false)

	case rbxfile.ValueInt64:
		g.value(int64(v))

	case rbxfile.ValueSharedString:
		g.value(string(v))

	case *bin.FormatModel:
		g.pushObject()
		g.field("Version", v.Version, true)
		g.field("Types", v.TypeCount, true)
		g.field("Instances", v.InstanceCount, true)
		g.pushArrayf("Chunks")
		for i, chunk := range v.Chunks {
			g.field("", chunk, i < len(v.Chunks)-1)
		}
		g.popArray(len(v.Chunks) > 0, false)
		g.popObject(false)

	case *bin.ChunkMeta:
		sig := v.Signature()
		g.pushObject()
		g.field("Signature", sig[:], true)
		g.field("Compressed", v.IsCompressed, true)
		g.pushArrayf("Value")
		for i, s := range v.Values {
			g.newline()
			g.pushObject()
			g.field("Key", s[0], true)
			g.field("Value", s[1], false)
			g.popObject(i < len(v.Values)-1)
		}
		g.popArray(len(v.Values) > 0, false)
		g.popObject(false)

	case *bin.ChunkSharedStrings:
		sig := v.Signature()
		g.pushObject()
		g.field("Signature", sig[:], true)
		g.field("Compressed", v.IsCompressed, true)
		g.field("Version", v.Version, true)
		g.pushArrayf("Values")
		for i, s := range v.Values {
			g.newline()
			g.pushObject()
			g.field("Hash", s.Hash[:], true)
			g.field("Value", s.Value, false)
			g.popObject(i < len(v.Values)-1)
		}
		g.popArray(len(v.Values) > 0, false)
		g.popObject(false)

	case *bin.ChunkInstance:
		sig := v.Signature()
		g.pushObject()
		g.field("Signature", sig[:], true)
		g.field("Compressed", v.IsCompressed, true)
		g.field("TypeID", v.TypeID, true)
		g.field("ClassName", v.ClassName, true)
		g.pushArrayf("InstanceIDs")
		for i, id := range v.InstanceIDs {
			g.field("", id, i < len(v.InstanceIDs)-1)
		}
		g.popArray(len(v.InstanceIDs) > 0, true)
		g.field("IsService", v.IsService, true)
		g.pushArrayf("GetService")
		for i, s := range v.GetService {
			g.field("", s, i < len(v.GetService)-1)
		}
		g.popArray(len(v.GetService) > 0, false)
		g.popObject(false)

	case *bin.ChunkProperty:
		sig := v.Signature()
		g.pushObject()
		g.field("Signature", sig[:], true)
		g.field("Compressed", v.IsCompressed, true)
		g.field("TypeID", v.TypeID, true)
		g.field("PropertyName", v.PropertyName, true)
		g.field("DataType", "0x"+strconv.FormatUint(uint64(v.DataType), 16)+" ("+v.DataType.String()+")", true)
		g.pushArrayf("Values")
		for i, prop := range v.Properties {
			g.field("", prop, i < len(v.Properties)-1)
		}
		g.popArray(len(v.Properties) > 0, false)
		g.popObject(false)

	case *bin.ChunkParent:
		sig := v.Signature()
		g.pushObject()
		g.field("Signature", sig[:], true)
		g.field("Compressed", v.IsCompressed, true)
		g.field("Version", v.Version, true)
		g.pushArrayf("Children")
		for i, child := range v.Children {
			g.field("", child, i < len(v.Children)-1)
		}
		g.popArray(len(v.Children) > 0, true)
		g.pushArrayf("Parents")
		for i, parent := range v.Parents {
			g.field("", parent, i < len(v.Parents)-1)
		}
		g.popArray(len(v.Parents) > 0, false)
		g.popObject(false)

	case *bin.ChunkEnd:
		sig := v.Signature()
		g.pushObject()
		g.field("Signature", sig[:], true)
		g.field("Compressed", v.IsCompressed, true)
		g.field("Content", v.Content, false)
		g.popObject(false)
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
	g.pushObject()
	g.field("Format", g.format, true)
	g.field("Output", g.structure, true)
	g.field("Data", v, false)
	g.popObject(false)
	return g
}

func (g *Golden) Bytes() []byte {
	return []byte(g.s.String())
}
