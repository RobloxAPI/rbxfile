// The rbxfile-stat command displays stats for a roblox file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/robloxapi/rbxfile"
	"github.com/robloxapi/rbxfile/rbxl"
)

const usage = `usage: rbxfile-stat [INPUT] [OUTPUT]

Reads a RBXL, RBXM, RBXLX, or RBXMX file from INPUT, and writes to OUTPUT
statistics for the file.

INPUT and OUTPUT are paths to files. If INPUT is "-" or unspecified, then stdin
is used. If OUTPUT is "-" or unspecified, then stdout is used. Warnings and
errors are written to stderr.
`

type PropLen struct {
	Class    string
	Property string
	Type     string
	Length   int
}

func (p PropLen) String() string {
	return fmt.Sprintf("%s.%s:%s(%d)", p.Class, p.Property, p.Type, p.Length)
}

type PropLenCount map[PropLen]int

func (p PropLenCount) MarshalJSON() ([]byte, error) {
	list := []PropLen{}
	for k := range p {
		list = append(list, k)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Length > list[j].Length
	})
	if len(list) > 20 {
		list = list[:20]
	}
	return json.Marshal(list)
}

type Stats struct {
	// Binary format data.
	Format rbxl.DecoderStats

	// Number of instances overall.
	InstanceCount int

	// Number of properties overall.
	PropertyCount int

	// Number of instances per class.
	ClassCount map[string]int

	// Number of properties per type.
	TypeCount map[string]int

	OptionalTypeCount map[string]int `json:",omitempty"`

	LargestProperties PropLenCount `json:",omitempty"`
}

const Okay = 0
const (
	Exit = 1 << iota
	SkipProperties
	SkipChildren
)

func walk(instances []*rbxfile.Instance, cb func(inst *rbxfile.Instance, property string, value rbxfile.Value) int) (ok bool) {
	for _, inst := range instances {
		status := cb(inst, "", nil)
		if status&Exit != 0 {
			return false
		}
		if status&SkipProperties == 0 {
			for property, value := range inst.Properties {
				status := cb(inst, property, value)
				if status&Exit != 0 {
					return false
				}
				if status&SkipProperties != 0 {
					break
				}
			}
		}
		if status&SkipChildren == 0 {
			if ok := walk(inst.Children, cb); !ok {
				return false
			}
		}
	}
	return true
}

func (s *Stats) Fill(root *rbxfile.Root) {
	if root == nil {
		return
	}

	s.PropertyCount = 0
	walk(root.Instances, func(inst *rbxfile.Instance, property string, value rbxfile.Value) int {
		if value == nil {
			return Okay
		}
		s.PropertyCount++
		return Okay
	})

	s.InstanceCount = 0
	s.ClassCount = map[string]int{}
	walk(root.Instances, func(inst *rbxfile.Instance, property string, value rbxfile.Value) int {
		s.InstanceCount++
		s.ClassCount[inst.ClassName]++
		return SkipProperties
	})

	s.TypeCount = map[string]int{}
	walk(root.Instances, func(inst *rbxfile.Instance, property string, value rbxfile.Value) int {
		if value == nil {
			return Okay
		}
		s.TypeCount[value.Type().String()]++
		return Okay
	})

	s.OptionalTypeCount = map[string]int{}
	walk(root.Instances, func(inst *rbxfile.Instance, property string, value rbxfile.Value) int {
		if value == nil {
			return Okay
		}
		opt, ok := value.(rbxfile.ValueOptional)
		if !ok {
			return Okay
		}
		s.OptionalTypeCount[opt.ValueType().String()]++
		return Okay
	})

	s.LargestProperties = PropLenCount{}
	walk(root.Instances, func(inst *rbxfile.Instance, property string, value rbxfile.Value) int {
		if value == nil {
			return Okay
		}
		var n int
		switch value := value.(type) {
		case rbxfile.ValueBinaryString:
			n = len(value)
		case rbxfile.ValueColorSequence:
			n = len(value)
		case rbxfile.ValueContent:
			n = len(value)
		case rbxfile.ValueNumberSequence:
			n = len(value)
		case rbxfile.ValueProtectedString:
			n = len(value)
		case rbxfile.ValueSharedString:
			n = len(value)
		case rbxfile.ValueString:
			n = len(value)
		default:
			return Okay
		}
		s.LargestProperties[PropLen{
			Class:    inst.ClassName,
			Property: property,
			Type:     value.Type().String(),
			Length:   n}]++
		return Okay
	})
}

func main() {
	var input io.Reader = os.Stdin
	var output io.Writer = os.Stdout

	flag.Usage = func() { fmt.Fprintf(flag.CommandLine.Output(), usage) }
	flag.Parse()
	args := flag.Args()
	if len(args) >= 1 && args[0] != "-" {
		in, err := os.Open(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("open input: %w", err))
			return
		}
		input = in
		defer in.Close()
	}
	if len(args) >= 2 && args[1] != "-" {
		out, err := os.Create(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("create output: %w", err))
			return
		}
		defer out.Close()
		defer func() {
			err := out.Sync()
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Errorf("sync output: %w", err))
				return
			}
		}()
		output = out
	}

	var stats Stats
	root, warn, err := rbxl.Decoder{Stats: &stats.Format}.Decode(input)
	if warn != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("decode warning: %w", warn))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("decode error: %w", warn))
	}

	stats.Fill(root)

	je := json.NewEncoder(output)
	je.SetEscapeHTML(false)
	je.SetIndent("", "\t")
	if err := je.Encode(stats); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("write error: %w", err))
	}
}
