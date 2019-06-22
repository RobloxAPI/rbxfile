package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/anaminus/but"
	"github.com/robloxapi/rbxfile/bin"
	"github.com/robloxapi/rbxfile/xml"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var update bool
var context uint
var color bool

type directives map[string]bool

func parseDirectives(input string) directives {
	d := directives{}
	for {
		ext := filepath.Ext(input)
		if ext == "" {
			break
		}
		d[ext[1:]] = true
		input = input[:len(input)-len(ext)]
	}
	return d
}

func openInput(input string) {
	file, err := os.Open(input)
	if err != nil {
		return
	}
	defer file.Close()

	format := strings.TrimPrefix(filepath.Ext(input), ".")
	directives := parseDirectives(input)

	var data interface{}
	switch format {
	case "rbxl":
		switch {
		case directives["format"]:
			doc := bin.FormatModel{}
			_, err = doc.ReadFrom(file)
			data = &doc
		case directives["model"]:
			fallthrough
		default:
			data, err = bin.DeserializePlace(file, nil)
		}
	case "rbxm":
		switch {
		case directives["format"]:
			doc := bin.FormatModel{}
			_, err = doc.ReadFrom(file)
			data = &doc
		case directives["model"]:
			fallthrough
		default:
			data, err = bin.DeserializeModel(file, nil)
		}
	case "rbxlx", "rbxmx":
		switch {
		case directives["format"]:
			doc := xml.Document{}
			_, err = doc.ReadFrom(file)
			data = &doc
		case directives["model"]:
			fallthrough
		default:
			data, err = xml.Deserialize(file, nil)
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
	diffGolden(input, int(context), spec, current)

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
