package xml

// Decoder adapted from the standard XML package.

// "DIFF" indicates a behavior that occurs in Roblox's codec, that is
// implemented differently in this codec.

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strconv"
)

// Tag represents a Roblox XML tag construct. Unlike standard XML, the content
// of a tag must consist of the following, in order:
//     1) An optional CData section.
//     2) A sequence of zero or more whitespace, which is ignored (usually newlines and indentation).
//     3) A sequence of zero or more characters indicating textual content of the tag.
//     4) A sequence of zero or more complete tags, with optional whitespace between each.
type Tag struct {
	// StartName is the name of the tag in the start tag.
	StartName string

	// EndName is the name of the tag in the end tag. If empty, this is
	// assumed to be equal to StartName.
	EndName string

	// The attributes of the tag.
	Attr []Attr

	// Empty indicates whether the tag has an empty-tag format. When encoding,
	// the tag will be written in the empty-tag format, and any content will
	// be ignored. When decoding, this value will be set if the decoded tag
	// has the empty-tag format.
	Empty bool

	// CData is a sequence of characters in a CDATA section. Only up to one
	// section is allowed, and must be the first element in the tag. A nil
	// array means that the tag does not contain a CDATA section.
	CData []byte

	// Text is the textual content of the tag.
	Text string

	// NoIndent indicates whether the tag contains prettifying whitespace,
	// which occurs between the tag's CData and Text, as well as between each
	// child tag.
	//
	// When decoding, this value is set to true if there is no whitespace of
	// any kind between the CData and Text. It will only be set if the decoder
	// has successfully detected global prefix and indent strings, but note
	// that these do not affect how the whitespace is detected.
	//
	// When encoding, this value determines whether the tag and its
	// descendants will be written with prettifying whitespace.
	NoIndent bool

	// Tags is a list of child tags within the tag.
	Tags []*Tag
}

// AttrValue returns the value of the first attribute of the given name, and
// whether or not it exists.
func (t Tag) AttrValue(name string) (value string, exists bool) {
	for _, a := range t.Attr {
		if a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}

// SetAttrValue sets the value of the first attribute of the given name, if it
// exists. If value is an empty string, then the attribute will be removed
// instead. If the attribute does not exist and value is not empty, then the
// attribute is added.
func (t *Tag) SetAttrValue(name, value string) {
	for i, a := range t.Attr {
		if a.Name == name {
			if value == "" {
				t.Attr = append(t.Attr[:i], t.Attr[i+1:]...)
			} else {
				a.Value = value
			}
			return
		}
	}
	if value == "" {
		return
	}
	t.Attr = append(t.Attr, Attr{Name: name, Value: value})
}

// NewRoot initializes a Tag containing values standard to a root tag.
// Optionally, Item tags can be given as arguments, which will be added to the
// root as sub-tags.
func NewRoot(items ...*Tag) *Tag {
	return &Tag{
		StartName: "roblox",
		Attr: []Attr{
			Attr{
				Name:  "xmlns:xmime",
				Value: "http://www.w3.org/2005/05/xmlmime",
			},
			Attr{
				Name:  "xmlns:xsi",
				Value: "http://www.w3.org/2001/XMLSchema-instance",
			},
			Attr{
				Name:  "xsi:noNamespaceSchemaLocation",
				Value: "http://www.roblox.com/roblox.xsd",
			},
			Attr{
				Name:  "version",
				Value: "4",
			},
		},
		Tags: items,
	}
}

// NewItem initializes an "Item" Tag representing a Roblox class.
func NewItem(class, referent string, properties ...*Tag) *Tag {
	return &Tag{
		StartName: "Item",
		Attr: []Attr{
			Attr{Name: "class", Value: class},
			Attr{Name: "referent", Value: referent},
		},
		Tags: []*Tag{
			&Tag{
				StartName: "Properties",
				Tags:      properties,
			},
		},
	}
}

// NewProp initializes a basic property tag representing a property in a
// Roblox class.
func NewProp(valueType, propName, value string) *Tag {
	return &Tag{
		StartName: valueType,
		Attr: []Attr{
			Attr{Name: "name", Value: propName},
		},
		Text:     value,
		NoIndent: true,
	}
}

// Attr represents an attribute of a tag.
type Attr struct {
	Name  string
	Value string
}

////////////////////////////////////////////////////////////////

// Document represents an entire XML document.
type Document struct {
	// Prefix is a string that appears at the start of each line in the
	// document.
	//
	// When encoding, the prefix is added after each newline. Newlines are
	// added automatically when either Prefix or Indent is not empty.
	//
	// When decoding, this value is set when indentation is detected in the
	// document. When detected, the value becomes any leading whitespace
	// before the root tag (at the start of the file). This only sets the
	// value; no attempt is made to validate any other prettifying whitespace.
	Prefix string

	// Indent is a string that indicates one level of indentation.
	//
	// When encoding, a sequence of indents appear after the Prefix, an amount
	// equal to the current nesting depth in the markup.
	//
	// When decoding, this value is set when detecting indentation. It is set
	// to the prettifying whitespace that occurs after the first newline and
	// prefix, which occurs between the root tag's CDATA and Text data. This
	// only sets the value; no attempt is made to validate any other
	// prettifying whitespace.
	Indent string

	// Suffix is a string that appears at the very end of the document. When
	// encoding, this string is appended to the end of the file, after the
	// root tag. When decoding, this value becomes any remaining text that
	// appears after the root tag.
	Suffix string

	// ExcludeRoot determines whether the root tag should be encoded. This can
	// be combined with Prefix to write documents in-line.
	ExcludeRoot bool

	// Root is the root tag in the document.
	Root *Tag

	// Warnings is a list of non-fatal problems that have occurred. This will
	// be cleared and populated when calling either ReadFrom and WriteTo.
	// Codecs may also clear and populate this when decoding or encoding.
	Warnings []error
}

// A SyntaxError represents a syntax error in the XML input stream.
type SyntaxError struct {
	Msg  string
	Line int
}

func (e *SyntaxError) Error() string {
	return "XML syntax error on line " + strconv.Itoa(e.Line) + ": " + e.Msg
}

type decoder struct {
	r        io.ByteReader
	buf      bytes.Buffer
	nextByte []byte
	doc      *Document
	n        int64
	err      error
	line     int
}

// Creates a SyntaxError with the current line number.
func (d *decoder) syntaxError(msg string) error {
	return &SyntaxError{Msg: msg, Line: d.line}
}

func (d *decoder) ignoreStartTag(err error) int {
	// Treat error as warning.
	d.doc.Warnings = append(d.doc.Warnings, err)
	// Read until end of start tag.
	for {
		b, ok := d.mustgetc()
		if !ok {
			return -1
		}
		if b == '>' {
			break
		}
	}
	return 0
}

//DIFF: Start tag parser has unexpected behavior that is difficult to
//pin-point.
func (d *decoder) decodeStartTag(tag *Tag) int {
	b, ok := d.getc()
	if !ok {
		return -1
	}

	if b != '<' {
		d.err = d.syntaxError("expected start tag")
		return -1
	}

	if b, ok = d.mustgetc(); !ok {
		return -1
	}
	if b == '/' {
		// </: End element; invalid
		d.err = d.syntaxError("unexpected end tag")
		return -1
	}

	// Must be an open element like <a href="foo">
	d.ungetc(b)

	if tag.StartName, ok = d.name(nameTag); !ok {
		return d.ignoreStartTag(d.syntaxError("expected element name after <"))
	}

	tag.Attr = make([]Attr, 0, 4)
	for {
		d.space()
		if b, ok = d.mustgetc(); !ok {
			return -1
		}
		if b == '/' {
			tag.Empty = true
			if b, ok = d.mustgetc(); !ok {
				return -1
			}
			if b != '>' {
				return d.ignoreStartTag(d.syntaxError("expected /> in element"))
			}
			break
		}
		if b == '>' {
			break
		}
		d.ungetc(b)

		n := len(tag.Attr)
		if n >= cap(tag.Attr) {
			nattr := make([]Attr, n, 2*cap(tag.Attr))
			copy(nattr, tag.Attr)
			tag.Attr = nattr
		}
		tag.Attr = tag.Attr[0 : n+1]
		a := &tag.Attr[n]
		if a.Name, ok = d.name(nameAttr); !ok {
			return d.ignoreStartTag(d.syntaxError("expected attribute name in element"))
		}
		d.space()
		if b, ok = d.mustgetc(); !ok {
			return -1
		}
		if b != '=' {
			return d.ignoreStartTag(d.syntaxError("attribute name without = in element"))
		} else {
			d.space()
			data := d.attrval()
			if data == nil {
				return -1
			}
			a.Value = string(data)
		}
	}
	return 1
}

func (d *decoder) decodeCData(tag *Tag) bool {
	tag.CData = nil

	// attempt to read CData opener
	const opener = "<![CDATA["
	for i := 0; i < len(opener); i++ {
		if b, ok := d.getc(); !ok {
			return false
		} else if b != opener[i] {
			// optional; unget characters and return ok status
			d.ungetc(b)
			for j := i - 1; j >= 0; j-- {
				d.ungetc(opener[j])
			}
			return true
		}
	}

	// Have <![CDATA[.  Read text until ]]>.
	tag.CData = d.text(-1, true)
	if tag.CData == nil {
		return false
	}
	return true
}

func (d *decoder) decodeText(tag *Tag) bool {
	text := d.text(-1, false)
	if text == nil {
		tag.Text = ""
		return false
	}
	tag.Text = string(text)
	return true
}

func (d *decoder) decodeEndTag(tag *Tag) bool {
	b, ok := d.getc()
	if !ok {
		return false
	}

	if b != '<' {
		d.err = d.syntaxError("expected start tag")
		return false
	}

	if b, ok = d.mustgetc(); !ok {
		return false
	}
	if b != '/' {
		d.err = d.syntaxError("expected end tag")
		return false
	}

	// </: End element
	if tag.EndName, ok = d.name(nameTag); !ok {
		if d.err == nil {
			d.err = d.syntaxError("expected element name after </")
		}
		return false
	}
	d.space()
	if b, ok = d.mustgetc(); !ok {
		return false
	}
	if b != '>' {
		d.err = d.syntaxError("invalid characters between </" + tag.EndName + " and >")
		return false
	}
	return true
}

func (d *decoder) decodeTag(root bool) (tag *Tag, err error) {
	if d.err != nil {
		return nil, d.err
	}

	tag = new(Tag)
	noindent := false
	nocontent := true

	if root {
		// Attempt to detect prefix
		p := d.readSpace()
		if len(p) > 0 {
			// Store it for later. Prefix will be unset if no indentation is
			// detected.
			d.doc.Prefix = string(p)
		}
	}

	startTagState := d.decodeStartTag(tag)
	if startTagState < 0 {
		return nil, d.err
	}

	if root {
		if tag.StartName != "roblox" {
			d.err = d.syntaxError("no roblox tag")
			return nil, d.err
		}

		if v, ok := tag.AttrValue("version"); !ok {
			//DIFF: returns success, but no data is read
			d.err = d.syntaxError("version attribute not specified")
			return nil, d.err
		} else {
			n, err := strconv.ParseInt(v, 10, 32)
			if err != nil {
				d.err = d.syntaxError("no version number")
				return nil, d.err
			}
			if n < 4 {
				d.err = d.syntaxError("schemaVersionLoading<4")
				return nil, d.err
			}
		}
	}

	if tag.Empty {
		if startTagState == 0 {
			return nil, nil
		}
		return
	}

	if !d.decodeCData(tag) {
		return nil, d.err
	}
	if len(tag.CData) > 0 {
		nocontent = false
	}

	// prettifying whitespace
	if root {
		// Attempt to detect indentation by looking at the (usually ignored)
		// whitespace under the root tag after the CDATA.
		ind := d.readSpace()
		// Must contain a newline, otherwise it wouldn't be indentation.
		if i := bytes.IndexByte(ind, '\n'); i > -1 {
			if !bytes.HasPrefix(ind[i+1:], []byte(d.doc.Prefix)) {
				// If line does not begin with the prefix detected previously,
				// then assume that the whitespace is badly formed, and cease
				// detection.
				d.doc.Prefix = ""
			} else {
				// Found newline and prefix, all of the remaining whitespace
				// indicates one level of indentation.
				d.doc.Indent = string(ind[i+1+len(d.doc.Prefix):])
			}
		}
	} else {
		if d.doc.Prefix != "" || d.doc.Indent != "" {
			if len(d.readSpace()) == 0 {
				noindent = true
			}
		} else {
			d.space()
		}
	}

	if !d.decodeText(tag) {
		return nil, d.err
	}
	if len(tag.Text) > 0 {
		nocontent = false
	}

	for {
		// prettifying whitespace between tags
		d.space()

		b, ok := d.getc()
		if !ok {
			return nil, d.err
		}

		if b != '<' {
			d.err = d.syntaxError("expected tag")
			return nil, d.err
		}

		if b, ok = d.mustgetc(); !ok {
			return nil, d.err
		}
		if b == '/' {
			// </: End element
			d.ungetc('/')
			d.ungetc('<')

			if !d.decodeEndTag(tag) {
				return nil, d.err
			}

			break
		}

		// child tag
		d.ungetc(b)
		d.ungetc('<')

		subtag, err := d.decodeTag(false)
		if err != nil {
			return nil, err
		}
		if subtag != nil {
			tag.Tags = append(tag.Tags, subtag)
		}
	}
	if len(tag.Tags) > 0 {
		nocontent = false
	}

	if !nocontent {
		// Do not set NoIndent if the tag is empty.
		tag.NoIndent = noindent
	}

	if startTagState == 0 {
		// Ignore the entire tag.
		return nil, nil
	}

	return tag, nil
}

func (d *decoder) attrval() []byte {
	b, ok := d.mustgetc()
	if !ok {
		return nil
	}
	// Handle quoted attribute values
	if b == '"' {
		return d.text(int(b), false)
	}

	d.err = d.syntaxError("unquoted or missing attribute value in element")
	return nil
}

func (d *decoder) readSpace() []byte {
	d.buf.Reset()
	for {
		b, ok := d.getc()
		if !ok {
			return d.buf.Bytes()
		}
		if !isSpace(b) {
			d.ungetc(b)
			return d.buf.Bytes()
		}
		d.buf.WriteByte(b)
	}
	return d.buf.Bytes()
}

// Skip spaces if any
func (d *decoder) space() {
	for {
		b, ok := d.getc()
		if !ok {
			return
		}
		if !isSpace(b) {
			d.ungetc(b)
			return
		}
	}
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\r', '\n', '\t', '\f':
		return true
	default:
		return false
	}
}

// Read a single byte.
// If there is no byte to read, return ok==false
// and leave the error in d.err.
// Maintain line number.
func (d *decoder) getc() (b byte, ok bool) {
	if d.err != nil {
		return 0, false
	}

	if len(d.nextByte) > 0 {
		b, d.nextByte = d.nextByte[len(d.nextByte)-1], d.nextByte[:len(d.nextByte)-1]
	} else {
		b, d.err = d.r.ReadByte()
		if d.err != nil {
			return 0, false
		}
		d.n++
	}
	if b == '\n' {
		d.line++
	}

	return b, true
}

// Must read a single byte.
// If there is no byte to read,
// set d.err to SyntaxError("unexpected EOF")
// and return ok==false
func (d *decoder) mustgetc() (b byte, ok bool) {
	if b, ok = d.getc(); !ok {
		if d.err == io.EOF {
			d.err = d.syntaxError("unexpected EOF")
		}
	}
	return
}

// Unread a single byte.
func (d *decoder) ungetc(b byte) {
	if b == '\n' {
		d.line--
	}
	d.nextByte = append(d.nextByte, b)
}

var entity = map[string]int{
	"lt":   '<',
	"gt":   '>',
	"amp":  '&',
	"apos": '\'',
	"quot": '"',
}

// Read plain text section (XML calls it character data).
// If quote >= 0, we are in a quoted string and need to find the matching quote.
// If cdata == true, we are in a <![CDATA[ section and need to find ]]>.
// On failure return nil and leave the error in d.err.
func (d *decoder) text(quote int, cdata bool) []byte {
	var b0, b1 byte
	var trunc int
	d.buf.Reset()
Input:
	for {
		b, ok := d.getc()
		if !ok {
			if cdata {
				if d.err == io.EOF {
					d.err = d.syntaxError("unexpected EOF in CDATA section")
				}
				return nil
			}
			break Input
		}

		// <![CDATA[ section ends with ]]>.
		// It is an error for ]]> to appear in ordinary text.
		if b0 == ']' && b1 == ']' && b == '>' {
			if cdata {
				trunc = 2
				break Input
			}
			return nil
		}

		// Stop reading text if we see a <.
		if b == '<' && !cdata {
			if quote >= 0 {
				return nil
			}
			d.ungetc('<')
			break Input
		}
		if quote >= 0 && b == byte(quote) {
			break Input
		}
		//DIFF: incomplete entity (no semicolon) *inserts* semicolon at end of
		//text
		if b == '&' && !cdata {
			// Read escaped character expression up to semicolon.
			// XML in all its glory allows a document to define and use
			// its own character names with <!ENTITY ...> directives.
			// Parsers are required to recognize lt, gt, amp, apos, and quot
			// even if they have not been declared.
			before := d.buf.Len()
			d.buf.WriteByte('&')
			var ok bool
			var text string
			var haveText bool
			if b, ok = d.mustgetc(); !ok {
				return nil
			}
			if b == '#' {
				//DIFF: characters between valid characters and semicolon are
				//ignored
				d.buf.WriteByte(b)
				if b, ok = d.mustgetc(); !ok {
					return nil
				}
				base := 10
				if b == 'x' {
					//DIFF: ERROR: unable to parse hexidecimal character code
					base = 16
					d.buf.WriteByte(b)
					if b, ok = d.mustgetc(); !ok {
						return nil
					}
				}
				start := d.buf.Len()
				for '0' <= b && b <= '9' ||
					base == 16 && 'a' <= b && b <= 'f' ||
					base == 16 && 'A' <= b && b <= 'F' {
					d.buf.WriteByte(b)
					if b, ok = d.mustgetc(); !ok {
						return nil
					}
				}
				if b != ';' {
					//DIFF: if numeric entity does not end with a semicolon,
					//then the remaining text is truncated. Note: This may be
					//a sign that the text is parsed out first, then entities
					//are converted afterwards.
					d.ungetc(b)
				} else {
					s := string(d.buf.Bytes()[start:])
					d.buf.WriteByte(';')
					n, err := strconv.ParseUint(s, base, 64)
					//DIFF: numeric entitiy is parsed as int32 and converted
					//to a byte
					if err == nil && n <= 255 {
						text = string([]byte{byte(n)})
						haveText = true
					}
				}
			} else {
				d.ungetc(b)
				if !d.readName(nameEntity) {
					if d.err != nil {
						return nil
					}
					ok = false
				}
				if b, ok = d.mustgetc(); !ok {
					return nil
				}
				if b != ';' {
					d.ungetc(b)
				} else {
					name := d.buf.Bytes()[before+1:]
					d.buf.WriteByte(';')

					s := string(name)
					if r, ok := entity[s]; ok {
						text = string(r)
						haveText = true
					}
				}
			}

			if haveText {
				d.buf.Truncate(before)
				d.buf.Write([]byte(text))
				b0, b1 = 0, 0
				continue Input
			}

			b0, b1 = 0, 0
			continue Input
		}

		// We must rewrite unescaped \r and \r\n into \n.
		if b == '\r' {
			d.buf.WriteByte('\n')
		} else if b1 == '\r' && b == '\n' {
			// Skip \r\n--we already wrote \n.
		} else {
			d.buf.WriteByte(b)
		}

		b0, b1 = b1, b
	}
	buf := d.buf.Bytes()
	buf = buf[0 : len(buf)-trunc]

	data := make([]byte, len(buf))
	copy(data, buf)

	return data
}

// Get name: /first(first|second)*/
// Do not set d.err if the name is missing (unless unexpected EOF is received):
// let the caller provide better context.
func (d *decoder) name(typ int) (s string, ok bool) {
	d.buf.Reset()
	if !d.readName(typ) {
		return "", false
	}

	return d.buf.String(), true
}

// Read a name and append its bytes to d.buf.
// The name is delimited by any single-byte character not valid in names.
// All multi-byte characters are accepted; the caller must check their validity.
func (d *decoder) readName(typ int) (ok bool) {
	var b byte
	if b, ok = d.mustgetc(); !ok {
		return
	}
	if !isNameByte(b, typ) {
		d.ungetc(b)
		return false
	}
	d.buf.WriteByte(b)

	for {
		if b, ok = d.mustgetc(); !ok {
			return
		}
		if !isNameByte(b, typ) {
			d.ungetc(b)
			break
		}
		d.buf.WriteByte(b)
	}
	return true
}

const (
	nameTag = iota
	nameAttr
	nameEntity
)

func isNameByte(c byte, t int) bool {
	if '!' <= c && c <= '~' && c != '>' {
		switch t {
		case 1:
			// Attribute
			return c != '='
		case 2:
			// Entity
			return c != ';'
		}
		return true
	}
	return false
}

// ReadFrom decode data from r into the Document.
func (doc *Document) ReadFrom(r io.Reader) (n int64, err error) {
	if r == nil {
		return 0, errors.New("reader is nil")
	}

	doc.Prefix = ""
	doc.Indent = ""
	doc.Warnings = doc.Warnings[:0]

	d := &decoder{
		doc:      doc,
		nextByte: make([]byte, 0, 9),
		line:     1,
	}
	if rb, ok := r.(io.ByteReader); ok {
		d.r = rb
	} else {
		d.r = bufio.NewReader(r)
	}

	doc.Root, err = d.decodeTag(true)
	if err != nil {
		return d.n, err
	}

	d.buf.Reset()
	for {
		b, ok := d.getc()
		if !ok {
			break
		}
		d.buf.WriteByte(b)
	}
	doc.Suffix = d.buf.String()

	return d.n, nil
}

type encoder struct {
	*bufio.Writer
	d          *Document
	putNewline bool
	depth      int
	indentedIn bool
	n          int64
	err        error
}

func (e *encoder) encodeCData(tag *Tag) bool {
	if tag.CData == nil {
		return true
	}

	e.writeString("<![CDATA[")
	e.write(tag.CData)
	e.writeString("]]>")
	if !e.flush() {
		return false
	}
	return true
}

func (e *encoder) encodeText(tag *Tag) bool {
	e.escapeString(tag.Text, true)
	if !e.flush() {
		return false
	}
	return true
}

func (e *encoder) checkName(name string, typ int) bool {
	if len(name) == 0 {
		return false
	}
	for _, c := range []byte(name) {
		if !isNameByte(c, typ) {
			return false
		}
	}
	return true
}

func (e *encoder) encodeTag(tag *Tag, noTags bool, noindent bool) int {
	if e.err != nil {
		return -1
	}

	endName := tag.EndName

	if !noTags {
		if !e.checkName(tag.StartName, nameTag) {
			e.d.Warnings = append(e.d.Warnings, errors.New("ignored tag with malformed start name `"+tag.StartName+"`"))
			return 0
		}

		if !e.checkName(endName, nameTag) && endName != "" {
			endName = tag.StartName
			e.d.Warnings = append(e.d.Warnings, errors.New("tag with malformed end name `"+tag.EndName+"`, used start name instead"))
		}

		e.writeByte('<')
		e.writeString(tag.StartName)

		for _, attr := range tag.Attr {
			if !e.checkName(attr.Name, nameAttr) {
				e.d.Warnings = append(e.d.Warnings, errors.New("ignored attribute with malformed name `"+attr.Name+"`"))
				continue
			}
			e.writeByte(' ')
			e.writeString(attr.Name)
			e.writeByte('=')
			e.writeByte('"')
			e.escapeString(attr.Value, false)
			e.writeByte('"')
		}

		if tag.Empty {
			e.writeByte('/')
			e.writeByte('>')
			if !e.flush() {
				return -1
			}
			return 1
		}

		e.writeByte('>')
		if !e.flush() {
			return -1
		}
	}

	if !e.encodeCData(tag) {
		return -1
	}

	if !noindent && !tag.NoIndent {
		if len(tag.Tags) > 0 {
			if noTags {
				e.writeIndent(0, true)
			} else {
				e.writeIndent(1, false)
			}
		}
	}

	if !e.encodeText(tag) {
		return -1
	}

	for i, sub := range tag.Tags {
		if r := e.encodeTag(sub, false, noindent || tag.NoIndent); r < 0 {
			return -1
		} else if r == 0 {
			continue
		}
		if !noindent && !tag.NoIndent {
			if i == len(tag.Tags)-1 {
				if noTags {
					e.writeIndent(0, true)
				} else {
					e.writeIndent(-1, false)
				}
			} else {
				e.writeIndent(0, false)
			}
		}
	}

	if !noTags {
		e.writeByte('<')
		e.writeByte('/')
		if endName == "" {
			e.writeString(tag.StartName)
		} else {
			e.writeString(endName)
		}
		e.writeByte('>')

		if !e.flush() {
			return -1
		}
	}

	return 1
}

func (e *encoder) write(p []byte) bool {
	if e.err != nil {
		return false
	}
	n, err := e.Write(p)
	e.n += int64(n)
	if err != nil {
		e.err = err
		return false
	}
	return true
}

func (e *encoder) writeByte(b byte) bool {
	if e.err != nil {
		return false
	}
	if err := e.WriteByte(b); err != nil {
		e.err = err
		return false
	}
	e.n += 1
	return true
}

func (e *encoder) writeString(s string) bool {
	if e.err != nil {
		return false
	}
	n, err := e.WriteString(s)
	e.n += int64(n)
	if err != nil {
		e.err = err
		return false
	}
	return true
}

func (e *encoder) flush() bool {
	if e.err != nil {
		return false
	}
	if err := e.Flush(); err != nil {
		e.err = err
		return false
	}
	return true
}

func (e *encoder) writeIndent(depthDelta int, notag bool) {
	if len(e.d.Prefix) == 0 && len(e.d.Indent) == 0 {
		return
	}
	if depthDelta < 0 {
		e.depth--
	} else if depthDelta > 0 {
		e.depth++
	}
	if !notag {
		e.WriteByte('\n')
		if len(e.d.Prefix) > 0 {
			e.WriteString(e.d.Prefix)
		}
		if len(e.d.Indent) > 0 {
			for i := 0; i < e.depth; i++ {
				e.WriteString(e.d.Indent)
			}
		}
	}
}

var (
	esc_quot = []byte("&quot;")
	esc_apos = []byte("&apos;")
	esc_amp  = []byte("&amp;")
	esc_lt   = []byte("&lt;")
	esc_gt   = []byte("&gt;")
)

// escapeString writes to p the properly escaped XML equivalent of the plain
// text data s. If escapeLead is true, then leading whitespace will be
// escaped.
func (e *encoder) escapeString(s string, escapeLead bool) {
	var esc []byte
	last := 0
	bs := []byte(s)
	for i := 0; i < len(bs); {
		esc = nil
		b := bs[i]
		i++

		if escapeLead {
			if isSpace(b) {
				goto numbered
			}
			escapeLead = false
		}

		switch b {
		case '"':
			esc = esc_quot
		case '\'':
			esc = esc_apos
		case '&':
			esc = esc_amp
		case '<':
			esc = esc_lt
		case '>':
			esc = esc_gt
		default:
			if ' ' <= b && b <= '~' || b == '\n' || b == '\r' {
				// literal
				continue
			} else {
				goto numbered
			}
		}

	numbered:
		if esc == nil {
			n := []byte(strconv.FormatInt(int64(b), 10))
			esc = make([]byte, len(n)+3)
			esc[0] = '&'
			esc[1] = '#'
			copy(esc[2:], n)
			esc[len(esc)-1] = ';'
		}

		e.writeString(s[last : i-1])
		e.write(esc)
		last = i
	}
	e.writeString(s[last:])
}

// WriteTo encodes the Document as bytes to w.
func (d *Document) WriteTo(w io.Writer) (n int64, err error) {
	d.Warnings = d.Warnings[:0]

	e := &encoder{Writer: bufio.NewWriter(w), d: d}

	e.writeString(e.d.Prefix)

	if r := e.encodeTag(d.Root, d.ExcludeRoot, d.Root.NoIndent); r < 0 {
		return e.n, e.err
	}

	e.writeString(e.d.Suffix)
	e.flush()
	return e.n, e.err
}
