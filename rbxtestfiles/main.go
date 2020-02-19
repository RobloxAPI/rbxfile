package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/anaminus/but"
	"github.com/robloxapi/rbxfile/bin"
	"github.com/robloxapi/rbxfile/xml"
)

var update bool
var context uint
var color bool

type directives struct {
	pairs map[string]string
	flags map[string]bool
}

func parseDirectives(input string, r *bufio.Reader) *directives {
	d := directives{
		pairs: map[string]string{
			"format": strings.TrimPrefix(filepath.Ext(input), "."),
		},
		flags: map[string]bool{},
	}

loop:
	for {
		c, _, err := r.ReadRune()
		if err != nil {
			r.UnreadRune()
			break loop
		}
		switch c {
		case '#':
			dir, err := r.ReadString('\n')
			if err != nil && err != io.EOF {
				break loop
			}
			switch {
			case strings.HasSuffix(dir, "\r\n"):
				dir = dir[:len(dir)-2]
			case strings.HasSuffix(dir, "\n"):
				dir = dir[:len(dir)-1]
			}
			if i := strings.IndexByte(dir, ':'); i < 0 {
				flag := strings.TrimSpace(dir)
				if flag == "begin-content" {
					break loop
				}
				d.flags[flag] = true
			} else {
				key := strings.TrimSpace(dir[:i])
				val := strings.TrimSpace(dir[i+1:])
				d.pairs[key] = val
			}
			if err == io.EOF {
				break loop
			}
		default:
			r.UnreadRune()
			break loop
		}
	}
	return &d
}

func openInput(input string) {
	file, err := os.Open(input)
	if err != nil {
		return
	}
	defer file.Close()

	r := bufio.NewReader(file)
	directives := parseDirectives(input, r)
	format := directives.pairs["format"]

	var data interface{}
	switch format {
	case "rbxl":
		switch directives.pairs["output"] {
		case "format":
			doc := bin.FormatModel{}
			_, err = doc.ReadFrom(r)
			data = &doc
		case "model":
			fallthrough
		default:
			data, err = bin.DeserializePlace(r)
		}
	case "rbxm":
		switch directives.pairs["output"] {
		case "format":
			doc := bin.FormatModel{}
			_, err = doc.ReadFrom(r)
			data = &doc
		case "model":
			fallthrough
		default:
			data, err = bin.DeserializeModel(r)
		}
	case "rbxlx", "rbxmx":
		switch directives.pairs["output"] {
		case "format":
			doc := xml.Document{}
			_, err = doc.ReadFrom(r)
			data = &doc
		case "model":
			fallthrough
		default:
			data, err = xml.Deserialize(r)
		}
	default:
		return
	}

	g := &Golden{}
	if err != nil {
		g.Format(format, err)
	} else {
		g.Format(format, data)
	}

	gold := input + ".golden"
	spec, err := ioutil.ReadFile(gold)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	current := g.Bytes()
	if bytes.Equal(current, spec) {
		return
	}
	if but.IfError(diffGolden(input, int(context), spec, current), "diff spec with current") {
		return
	}

	if !update {
		return
	}
	fmt.Fprintf(os.Stderr, "Update %s? (y/N): ", filepath.Base(input))
	var s string
	fmt.Scanln(&s)
	if strings.ToLower(s) != "y" {
		return
	}
	but.IfError(ioutil.WriteFile(gold, current, 0666))
}

func checkFile(inputs []string, dir string, info os.FileInfo, emit bool) []string {
	switch name := info.Name(); {
	default:
		inputs = append(inputs, filepath.Join(dir, name))
	case info.IsDir():
		path := filepath.Join(dir, name)
		infos, err := ioutil.ReadDir(path)
		if err != nil {
			if emit {
				but.IfError(err)
			}
			break
		}
		for _, sub := range infos {
			inputs = checkFile(inputs, path, sub, false)
		}
	case len(name) == 0:
	case name[0] == '.':
	case filepath.Ext(name) == ".golden":
	}
	return inputs
}

func main() {
	flag.BoolVar(&update, "update", false, "Update golden files.")
	flag.UintVar(&context, "context", 3, "Amount of context for diffs.")
	flag.BoolVar(&color, "color", false, "Use terminal colors.")
	flag.Parse()

	var inputs []string
	for _, file := range flag.Args() {
		info, err := os.Stat(file)
		if but.IfError(err) {
			continue
		}
		inputs = checkFile(inputs, filepath.Dir(file), info, true)
	}
	for _, input := range inputs {
		openInput(input)
	}
}
