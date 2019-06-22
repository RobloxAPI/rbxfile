package main

import (
	"fmt"
	"strconv"
	"strings"
)

func intlen(i int) int {
	n := 1
	if i >= 100000000 {
		n += 8
		i /= 100000000
	}
	if i >= 10000 {
		n += 4
		i /= 10000
	}
	if i >= 100 {
		n += 2
		i /= 100
	}
	if i >= 10 {
		n += 1
	}
	return n
}

func setLine(buf []byte, l int, op byte) []byte {
	if l < 0 {
		for i := range buf {
			buf[i] = '.'
		}
		return buf
	}
	n := len(buf) - 1 - intlen(l)
	for i := 0; i < n; i++ {
		buf[i] = ' '
	}
	strconv.AppendUint(buf[n:n], uint64(l), 10)
	buf[len(buf)-1] = op
	return buf
}

func diffGolden(input string, context int, curr, spec []byte) {
	currLines := strings.Split(string(curr), "\n")
	specLines := strings.Split(string(spec), "\n")
	chunks := DiffChunks(currLines, specLines)

	type lineData struct {
		line int
		op   byte
		text string
	}

	var lines []lineData
	{
		var prev []string
		var prevLine int
		for _, c := range chunks {
			if len(c.Added)+len(c.Deleted) == 0 {
				i := len(c.Equal) - context
				if i < 0 {
					i = 0
				}
				prev = c.Equal[i:]
				prevLine = c.EqualLine + i
				continue
			}
			for i, line := range prev {
				lines = append(lines, lineData{
					line: prevLine + i,
					op:   ' ',
					text: line,
				})
			}
			for i, line := range c.Deleted {
				lines = append(lines, lineData{
					line: c.DeletedLine + i,
					op:   '-',
					text: line,
				})
			}
			for i, line := range c.Added {
				lines = append(lines, lineData{
					line: c.AddedLine + i,
					op:   '+',
					text: line,
				})
			}
			prev = nil
			for i, line := range c.Equal {
				if i < context {
					lines = append(lines, lineData{
						line: c.EqualLine + i,
						op:   ' ',
						text: line,
					})
					continue
				}
				if j := len(c.Equal) - context; i < j {
					i = j
				}
				prev = c.Equal[i:]
				prevLine = c.EqualLine + i
				break
			}
		}
	}

	if len(lines) == 0 {
		return
	}
	// sort.Slice(lines, func(i, j int) bool {
	// 	if lines[i].line == lines[j].line {
	// 		return lines[i].op > lines[j].op
	// 	}
	// 	return lines[i].line < lines[j].line
	// })

	var max int
	for _, line := range lines {
		if line.line > max {
			max = line.line
		}
	}

	buf := make([]byte, intlen(max)+1)

	var b strings.Builder
	b.WriteString("--- ")
	b.WriteString(input)
	b.WriteString(".golden\n")
	b.WriteString("+++ ")
	b.WriteString(input)
	b.WriteByte('\n')
	var prev lineData
	for _, line := range lines {
		if color {
			switch line.op {
			case '+':
				b.WriteString("\x1b[33m")
			case '-':
				b.WriteString("\x1b[34m")
			default:
				b.WriteString("\x1b[0m")
			}
		}
		if line.op == ' ' && prev.op == ' ' && line.line-1 != prev.line {
			b.Write(setLine(buf[:len(buf)-1], -1, 0))
			b.WriteByte('\n')
		}
		b.Write(setLine(buf, line.line, line.op))
		b.WriteString(line.text)
		b.WriteByte('\n')
		prev = line
	}
	if color {
		b.WriteString("\x1b[0m")
	}
	fmt.Println(b.String())
}
