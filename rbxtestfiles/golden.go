package main

import (
	"github.com/robloxapi/rbxfile"
	"github.com/robloxapi/rbxfile/bin"
	"github.com/robloxapi/rbxfile/xml"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Golden struct {
	s         strings.Builder
	buf       []byte
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

func (g *Golden) value(v string) *Golden {
	g.s.Write(g.lead)
	g.s.WriteString(v)
	g.s.WriteByte('\n')
	return g
}

func (g *Golden) pushf(f string) *Golden {
	if f != "" {
		g.value(f)
		g.push()
	}
	return g
}
func (g *Golden) popf(f string) *Golden {
	if f != "" {
		g.pop()
	}
	return g
}

func (g *Golden) field(f, v string) *Golden {
	if f == "" {
		g.value(v)
		return g
	}
	g.s.Write(g.lead)
	g.s.WriteString(f)
	g.s.WriteString(": ")
	g.s.WriteString(v)
	g.s.WriteByte('\n')
	return g
}

func (g *Golden) bool(f string, v bool) *Golden {
	if v {
		g.field(f, "true")
		return g
	}
	g.field(f, "false")
	return g
}

func (g *Golden) int(f string, v interface{}) *Golden {
	var n int64
	switch v := v.(type) {
	case int:
		n = int64(v)
	case int8:
		n = int64(v)
	case int16:
		n = int64(v)
	case int32:
		n = int64(v)
	case int64:
		n = v
	default:
		panic("expected signed integer (got " + reflect.TypeOf(v).String() + ")")
	}
	g.field(f, strconv.FormatInt(n, 10))
	return g
}

func (g *Golden) uint(f string, v interface{}) *Golden {
	var n uint64
	switch v := v.(type) {
	case uint:
		n = uint64(v)
	case uint8:
		n = uint64(v)
	case uint16:
		n = uint64(v)
	case uint32:
		n = uint64(v)
	case uint64:
		n = v
	default:
		panic("expected unsigned integer (got " + reflect.TypeOf(v).String() + ")")
	}
	g.field(f, strconv.FormatUint(n, 10))
	return g
}

func (g *Golden) float(f string, v interface{}) *Golden {
	var s string
	switch v := v.(type) {
	case float32:
		s = strconv.FormatFloat(float64(v), 'g', 9, 32)
	case float64:
		s = strconv.FormatFloat(float64(v), 'g', 17, 64)
	default:
		panic("expected float (got " + reflect.TypeOf(v).String() + ")")
	}
	// s = strings.TrimRight(s, "0")
	g.field(f, s)
	return g

	g.s.Write(g.lead)
	if f != "" {
		g.s.WriteString(f)
		g.s.WriteString(": ")
	}

	var bits uint64
	var mantbits uint
	var expbits uint
	switch v := v.(type) {
	case float32:
		bits = uint64(math.Float32bits(v))
		mantbits = 23
		expbits = 8
	case float64:
		bits = math.Float64bits(v)
		mantbits = 52
		expbits = 11
	default:
		panic("expected float (got " + reflect.TypeOf(v).String() + ")")
	}

	neg := int(bits>>(expbits+mantbits)) != 0
	exp := (bits >> mantbits) & (1<<expbits - 1)
	mant := bits & (uint64(1)<<mantbits - 1)

	if exp == 1<<expbits-1 {
		switch {
		case mant != 0:
			g.s.WriteString("NaN")
		case neg:
			g.s.WriteString("-Inf")
		default:
			g.s.WriteString("+Inf")
		}
		goto finish
	}

	if neg && (exp != 0 || mant != 0) {
		g.s.WriteByte('-')
	} else {
		g.s.WriteByte('+')
	}
	g.buf = strconv.AppendUint(g.buf[:0], exp, 10)
	g.s.Write(g.buf)
	g.s.WriteByte('_')
	g.buf = strconv.AppendUint(g.buf[:0], mant, 10)
	g.s.Write(g.buf)

finish:
	g.s.WriteByte('\n')
	return g
}

func (g *Golden) bytes(f string, v []byte, width int) *Golden {
	if width <= 0 {
		width = 16
	}
	g.pushf(f)
	g.field("Length", strconv.Itoa(len(v)))
	for j := 0; j < len(v); j += width {
		g.s.Write(g.lead)
		g.s.WriteString("| ")
		for i := j; i < j+width; {
			if i < len(v) {
				s := strconv.FormatUint(uint64(v[i]), 16)
				if len(s) == 1 {
					g.s.WriteString("0")
				}
				g.s.WriteString(s)
			} else if len(v) < width {
				break
			} else {
				g.s.WriteString("  ")
			}
			i++
			if i%8 == 0 && i < j+width {
				g.s.WriteString("  ")
			} else {
				g.s.WriteString(" ")
			}
		}
		g.s.WriteString("|")
		n := len(v)
		if j+width < n {
			n = j + width
		}
		for i := j; i < n; i++ {
			if 32 <= v[i] && v[i] <= 126 {
				g.s.WriteRune(rune(v[i]))
			} else {
				g.s.WriteByte('.')
			}
		}
		g.s.WriteString("|\n")
	}
	g.popf(f)
	return g
}

func (g *Golden) string(f string, v string) *Golden {
	for _, r := range []rune(v) {
		switch r {
		case '\t', '\n':
			continue
		}
		if !unicode.IsGraphic(r) {
			return g.bytes(f, []byte(v), 16)
		}
	}
	g.s.Write(g.lead)
	if f != "" {
		g.s.WriteString(f)
		g.s.WriteString(": ")
	}
	g.s.WriteByte('"')
	r := strings.NewReplacer(
		"\"", "\\\"",
		"\\", "\\\\",
		"\t", "\\t",
		"\n", "\\n",
	)
	r.WriteString(&g.s, v)
	g.s.WriteByte('"')
	g.s.WriteByte('\n')
	return g
}

func (g *Golden) ref(f string, v *rbxfile.Instance) *Golden {
	if v == nil {
		g.field(f, "nil")
		return g
	}
	g.int(f, g.refs[v])
	return g
}

func recurseRefs(refs map[*rbxfile.Instance]int, instances []*rbxfile.Instance) {
	for _, inst := range instances {
		refs[inst] = len(refs)
		recurseRefs(refs, inst.Children)
	}
}

func (g *Golden) Field(name string, v interface{}) *Golden {
	switch v := v.(type) {
	default:
		g.field(name, "<UNKNOWN:"+reflect.TypeOf(v).String()+">")

	case error:
		g.field(name, v.Error())

	case *rbxfile.Root:
		// Prepopulate ref table.
		g.refs = map[*rbxfile.Instance]int{}
		recurseRefs(g.refs, v.Instances)

		g.pushf(name)
		meta := make([]string, 0, len(v.Metadata))
		for k := range v.Metadata {
			meta = append(meta, k)
		}
		sort.Strings(meta)
		g.value("Metadata")
		g.push()
		for i, k := range meta {
			g.int("", i)
			g.push()
			g.string("Key", k)
			g.string("Value", v.Metadata[k])
			g.pop()
		}
		g.pop()
		g.value("Instances")
		g.push()
		for i, inst := range v.Instances {
			g.Field(strconv.Itoa(i), inst)
		}
		g.pop()
		g.popf(name)

	case *rbxfile.Instance:
		g.pushf(name)
		g.string("ClassName", v.ClassName)
		g.bool("IsService", v.IsService)
		g.ref("Reference", v)
		g.pushf("Properties")
		props := make([]string, 0, len(v.Properties))
		for name := range v.Properties {
			props = append(props, name)
		}
		sort.Strings(props)
		for i, name := range props {
			value := v.Properties[name]
			g.pushf(strconv.Itoa(i))
			g.string("Name", name)
			g.string("Type", value.Type().String())
			g.Field("Value", value)
			g.pop()
		}
		g.pop()
		g.pushf("Children")
		for i, child := range v.Children {
			g.Field(strconv.Itoa(i), child)
		}
		g.pop()
		g.popf(name)

	case rbxfile.ValueString:
		g.string(name, string(v))

	case rbxfile.ValueBinaryString:
		g.bytes(name, []byte(v), 16)

	case rbxfile.ValueProtectedString:
		g.string(name, string(v))

	case rbxfile.ValueContent:
		g.string(name, string(v))

	case rbxfile.ValueBool:
		g.bool(name, bool(v))

	case rbxfile.ValueInt:
		g.int(name, int64(v))

	case rbxfile.ValueFloat:
		g.float(name, float32(v))

	case rbxfile.ValueDouble:
		g.float(name, float64(v))

	case rbxfile.ValueUDim:
		g.pushf(name)
		g.float("Scale", v.Scale)
		g.int("Offset", v.Offset)
		g.popf(name)

	case rbxfile.ValueUDim2:
		g.pushf(name)
		g.Field("X", v.X)
		g.Field("Y", v.Y)
		g.popf(name)

	case rbxfile.ValueRay:
		g.pushf(name)
		g.Field("Origin", v.Origin)
		g.Field("Direction", v.Direction)
		g.popf(name)

	case rbxfile.ValueFaces:
		g.pushf(name)
		g.bool("Right", v.Right)
		g.bool("Top", v.Top)
		g.bool("Back", v.Back)
		g.bool("Left", v.Left)
		g.bool("Bottom", v.Bottom)
		g.bool("Front", v.Front)
		g.popf(name)

	case rbxfile.ValueAxes:
		g.pushf(name)
		g.bool("X", v.X)
		g.bool("Y", v.Y)
		g.bool("Z", v.Z)
		g.popf(name)

	case rbxfile.ValueBrickColor:
		g.uint(name, uint32(v))

	case rbxfile.ValueColor3:
		g.pushf(name)
		g.float("R", v.R)
		g.float("G", v.G)
		g.float("B", v.B)
		g.popf(name)

	case rbxfile.ValueVector2:
		g.pushf(name)
		g.float("X", v.X)
		g.float("Y", v.Y)
		g.popf(name)

	case rbxfile.ValueVector3:
		g.pushf(name)
		g.float("X", v.X)
		g.float("Y", v.Y)
		g.float("Z", v.Z)
		g.popf(name)

	case rbxfile.ValueCFrame:
		g.pushf(name)
		g.Field("Position", v.Position)
		g.pushf("Rotation")
		g.float("R00", v.Rotation[0])
		g.float("R01", v.Rotation[1])
		g.float("R02", v.Rotation[2])
		g.float("R10", v.Rotation[3])
		g.float("R11", v.Rotation[4])
		g.float("R12", v.Rotation[5])
		g.float("R20", v.Rotation[6])
		g.float("R21", v.Rotation[7])
		g.float("R22", v.Rotation[8])
		g.pop()
		g.popf(name)

	case rbxfile.ValueToken:
		g.uint(name, uint64(v))

	case rbxfile.ValueReference:
		g.ref("Value", v.Instance)

	case rbxfile.ValueVector3int16:
		g.pushf(name)
		g.int("X", v.X)
		g.int("Y", v.Y)
		g.int("Z", v.Z)
		g.popf(name)

	case rbxfile.ValueVector2int16:
		g.pushf(name)
		g.int("X", v.X)
		g.int("Y", v.Y)
		g.popf(name)

	case rbxfile.ValueNumberSequence:
		g.pushf(name)
		for i, k := range v {
			g.int("", i)
			g.push()
			g.float("Time", k.Time)
			g.float("Value", k.Value)
			g.float("Envelope", k.Envelope)
			g.pop()
		}
		g.popf(name)

	case rbxfile.ValueColorSequence:
		g.pushf(name)
		for i, k := range v {
			g.int("", i)
			g.push()
			g.float("Time", k.Time)
			g.Field("Value", k.Value)
			g.float("Envelope", k.Envelope)
			g.pop()
		}
		g.popf(name)

	case rbxfile.ValueNumberRange:
		g.pushf(name)
		g.float("Min", v.Min)
		g.float("Max", v.Max)
		g.popf(name)

	case rbxfile.ValueRect2D:
		g.pushf(name)
		g.Field("Min", v.Min)
		g.Field("Max", v.Max)
		g.popf(name)

	case rbxfile.ValuePhysicalProperties:
		g.pushf(name)
		g.bool("CustomPhysics", v.CustomPhysics)
		g.float("Density", v.Density)
		g.float("Friction", v.Friction)
		g.float("Elasticity", v.Elasticity)
		g.float("FrictionWeight", v.FrictionWeight)
		g.float("ElasticityWeight", v.ElasticityWeight)
		g.popf(name)

	case rbxfile.ValueColor3uint8:
		g.pushf(name)
		g.uint("R", v.R)
		g.uint("G", v.G)
		g.uint("B", v.B)
		g.popf(name)

	case rbxfile.ValueInt64:
		g.int(name, int64(v))

	case rbxfile.ValueSharedString:
		g.string(name, string(v))

	case *bin.FormatModel:
		g.pushf(name)
		g.uint("Version", v.Version)
		g.uint("Types", v.TypeCount)
		g.uint("Instances", v.InstanceCount)
		g.value("Chunks")
		g.push()
		for i, chunk := range v.Chunks {
			g.Field(strconv.Itoa(i), chunk)
		}
		g.pop()
		g.popf(name)

	case *bin.ChunkMeta:
		g.pushf(name)
		sig := v.Signature()
		g.bytes("Signature", sig[:], len(sig))
		g.bool("Compressed", v.IsCompressed)
		g.pushf("Values")
		for i, v := range v.Values {
			g.pushf(strconv.Itoa(i))
			g.string("Key", v[0])
			g.string("Value", v[1])
			g.pop()
		}
		g.pop()
		g.popf(name)

	case *bin.ChunkSharedStrings:
		g.pushf(name)
		sig := v.Signature()
		g.bytes("Signature", sig[:], len(sig))
		g.bool("Compressed", v.IsCompressed)
		g.uint("Version", v.Version)
		g.pushf("Values")
		for i, v := range v.Values {
			g.pushf(strconv.Itoa(i))
			g.bytes("Hash", v.Hash[:], 16)
			g.bytes("Value", v.Value, 16)
			g.pop()
		}
		g.pop()
		g.popf(name)

	case *bin.ChunkInstance:
		g.pushf(name)
		sig := v.Signature()
		g.bytes("Signature", sig[:], len(sig))
		g.bool("Compressed", v.IsCompressed)
		g.int("TypeID", v.TypeID)
		g.string("ClassName", v.ClassName)
		g.pushf("InstanceIDs")
		for i, v := range v.InstanceIDs {
			g.int(strconv.Itoa(i), v)
		}
		g.pop()
		g.bool("IsService", v.IsService)
		g.pushf("GetService")
		for i, v := range v.GetService {
			g.uint(strconv.Itoa(i), v)
		}
		g.pop()
		g.popf(name)

	case *bin.ChunkProperty:
		g.pushf(name)
		sig := v.Signature()
		g.bytes("Signature", sig[:], len(sig))
		g.bool("Compressed", v.IsCompressed)
		g.int("TypeID", v.TypeID)
		g.string("PropertyName", v.PropertyName)
		g.string("DataType", "0x"+strconv.FormatUint(uint64(v.DataType), 16)+" ("+v.DataType.String()+")")
		g.pushf("Values")
		for i, prop := range v.Properties {
			g.Field(strconv.Itoa(i), prop)
		}
		g.pop()
		g.popf(name)

	case *bin.ChunkParent:
		g.pushf(name)
		sig := v.Signature()
		g.bytes("Signature", sig[:], len(sig))
		g.bool("Compressed", v.IsCompressed)
		g.uint("Version", v.Version)
		g.pushf("Children")
		for i, v := range v.Children {
			g.int(strconv.Itoa(i), v)
		}
		g.pop()
		g.pushf("Parents")
		for i, v := range v.Parents {
			g.int(strconv.Itoa(i), v)
		}
		g.pop()
		g.popf(name)

	case *bin.ChunkEnd:
		g.pushf(name)
		sig := v.Signature()
		g.bytes("Signature", sig[:], len(sig))
		g.bool("Compressed", v.IsCompressed)
		g.bytes("Content", v.Content, 16)
		g.popf(name)
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
	g.field("Format", g.format)
	g.field("Struct", g.structure)
	g.Field("Data", v)
	return g
}

func (g *Golden) Bytes() []byte {
	return []byte(g.s.String())
}
