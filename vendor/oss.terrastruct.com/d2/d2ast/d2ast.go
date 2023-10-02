// d2ast implements the d2 language's abstract syntax tree.
//
// Special characters to think about in parser:
//   #
//   """
//   ;
//   []
//   {}
//   |
//   $
//   '
//   "
//   \
//   :
//   .
//   --
//   <>
//   *
//   &
//   ()
package d2ast

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"oss.terrastruct.com/util-go/xdefer"
)

// Node is the base interface implemented by all d2 AST nodes.
// TODO: add error node for autofmt of incomplete AST
type Node interface {
	node()

	// Type returns the user friendly name of the node.
	Type() string

	// GetRange returns the range a node occupies in its file.
	GetRange() Range

	// TODO: add Children() for walking AST
	// Children() []Node
}

var _ Node = &Comment{}
var _ Node = &BlockComment{}

var _ Node = &Null{}
var _ Node = &Boolean{}
var _ Node = &Number{}
var _ Node = &UnquotedString{}
var _ Node = &DoubleQuotedString{}
var _ Node = &SingleQuotedString{}
var _ Node = &BlockString{}
var _ Node = &Substitution{}

var _ Node = &Array{}
var _ Node = &Map{}
var _ Node = &Key{}
var _ Node = &KeyPath{}
var _ Node = &Edge{}
var _ Node = &EdgeIndex{}

// Range represents a range between Start and End in Path.
// It's also used in the d2parser package to represent the range of an error.
//
// note: See docs on Position.
//
// It has a custom JSON string encoding with encoding.TextMarshaler and
// encoding.TextUnmarshaler to keep it compact as the JSON struct encoding is too verbose,
// especially with json.MarshalIndent.
//
// It looks like path,start-end
type Range struct {
	Path  string
	Start Position
	End   Position
}

var _ fmt.Stringer = Range{}
var _ encoding.TextMarshaler = Range{}
var _ encoding.TextUnmarshaler = &Range{}

func MakeRange(s string) Range {
	var r Range
	_ = r.UnmarshalText([]byte(s))
	return r
}

// String returns a string representation of the range including only the path and start.
//
// If path is empty, it will be omitted.
//
// The format is path:start
func (r Range) String() string {
	var s strings.Builder
	if r.Path != "" {
		s.WriteString(r.Path)
		s.WriteByte(':')
	}

	s.WriteString(r.Start.String())
	return s.String()
}

// OneLine returns true if the Range starts and ends on the same line.
func (r Range) OneLine() bool {
	return r.Start.Line == r.End.Line
}

// See docs on Range.
func (r Range) MarshalText() ([]byte, error) {
	start, _ := r.Start.MarshalText()
	end, _ := r.End.MarshalText()
	return []byte(fmt.Sprintf("%s,%s-%s", r.Path, start, end)), nil
}

// See docs on Range.
func (r *Range) UnmarshalText(b []byte) (err error) {
	defer xdefer.Errorf(&err, "failed to unmarshal Range from %q", b)

	i := bytes.LastIndexByte(b, '-')
	if i == -1 {
		return errors.New("missing End field")
	}
	end := b[i+1:]
	b = b[:i]

	i = bytes.LastIndexByte(b, ',')
	if i == -1 {
		return errors.New("missing Start field")
	}
	start := b[i+1:]
	b = b[:i]

	r.Path = string(b)
	err = r.Start.UnmarshalText(start)
	if err != nil {
		return err
	}
	return r.End.UnmarshalText(end)
}

// Position represents a line:column and byte position in a file.
//
// note: Line and Column are zero indexed.
// note: Column and Byte are UTF-8 byte indexes unless byUTF16 was passed to Position.Advance in
//       which they are UTF-16 code unit indexes.
//       If intended for Javascript consumption like in the browser or via LSP, byUTF16 is
//       set to true.
type Position struct {
	Line   int
	Column int
	Byte   int
}

var _ fmt.Stringer = Position{}
var _ encoding.TextMarshaler = Position{}
var _ encoding.TextUnmarshaler = &Position{}

// String returns a line:column representation of the position suitable for error messages.
//
// note: Should not normally be used directly, see Range.String()
func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line+1, p.Column+1)
}

// See docs on Range.
func (p Position) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%d:%d:%d", p.Line, p.Column, p.Byte)), nil
}

// See docs on Range.
func (p *Position) UnmarshalText(b []byte) (err error) {
	defer xdefer.Errorf(&err, "failed to unmarshal Position from %q", b)

	fields := bytes.Split(b, []byte{':'})
	if len(fields) != 3 {
		return errors.New("expected three fields")
	}

	p.Line, err = strconv.Atoi(string(fields[0]))
	if err != nil {
		return err
	}
	p.Column, err = strconv.Atoi(string(fields[1]))
	if err != nil {
		return err
	}
	p.Byte, err = strconv.Atoi(string(fields[2]))
	return err
}

// From copies src into p. It's used in the d2parser package to set a node's Range.End to
// the parser's current pos on all return paths with defer.
func (p *Position) From(src *Position) {
	*p = *src
}

// Advance advances p's Line, Column and Byte by r and returns
// the new Position.
// Set byUTF16 to advance the position as though r represents
// a UTF-16 codepoint.
func (p Position) Advance(r rune, byUTF16 bool) Position {
	size := utf8.RuneLen(r)
	if byUTF16 {
		size = 1
		r1, r2 := utf16.EncodeRune(r)
		if r1 != '\uFFFD' && r2 != '\uFFFD' {
			size = 2
		}
	}

	if r == '\n' {
		p.Line++
		p.Column = 0
	} else {
		p.Column += size
	}
	p.Byte += size

	return p
}

func (p Position) Subtract(r rune, byUTF16 bool) Position {
	size := utf8.RuneLen(r)
	if byUTF16 {
		size = 1
		r1, r2 := utf16.EncodeRune(r)
		if r1 != '\uFFFD' && r2 != '\uFFFD' {
			size = 2
		}
	}

	if r == '\n' {
		panic("d2ast: cannot subtract newline from Position")
	} else {
		p.Column -= size
	}
	p.Byte -= size

	return p
}

func (p Position) SubtractString(s string, byUTF16 bool) Position {
	for _, r := range s {
		p = p.Subtract(r, byUTF16)
	}
	return p
}

// MapNode is implemented by nodes that may be children of Maps.
type MapNode interface {
	Node
	mapNode()
}

var _ MapNode = &Comment{}
var _ MapNode = &BlockComment{}
var _ MapNode = &Key{}
var _ MapNode = &Substitution{}

// ArrayNode is implemented by nodes that may be children of Arrays.
type ArrayNode interface {
	Node
	arrayNode()
}

// See Value for the rest.
var _ ArrayNode = &Comment{}
var _ ArrayNode = &BlockComment{}
var _ ArrayNode = &Substitution{}

// Value is implemented by nodes that may be values of a key.
type Value interface {
	ArrayNode
	value()
}

// See Scalar for rest.
var _ Value = &Array{}
var _ Value = &Map{}

// Scalar is implemented by nodes that represent scalar values.
type Scalar interface {
	Value
	scalar()
	ScalarString() string
}

// See String for rest.
var _ Scalar = &Null{}
var _ Scalar = &Boolean{}
var _ Scalar = &Number{}

// String is implemented by nodes that represent strings.
type String interface {
	Scalar
	SetString(string)
	Copy() String
	_string()
}

var _ String = &UnquotedString{}
var _ String = &SingleQuotedString{}
var _ String = &DoubleQuotedString{}
var _ String = &BlockString{}

func (c *Comment) node()            {}
func (c *BlockComment) node()       {}
func (n *Null) node()               {}
func (b *Boolean) node()            {}
func (n *Number) node()             {}
func (s *UnquotedString) node()     {}
func (s *DoubleQuotedString) node() {}
func (s *SingleQuotedString) node() {}
func (s *BlockString) node()        {}
func (s *Substitution) node()       {}
func (a *Array) node()              {}
func (m *Map) node()                {}
func (k *Key) node()                {}
func (k *KeyPath) node()            {}
func (e *Edge) node()               {}
func (i *EdgeIndex) node()          {}

func (c *Comment) Type() string            { return "comment" }
func (c *BlockComment) Type() string       { return "block comment" }
func (n *Null) Type() string               { return "null" }
func (b *Boolean) Type() string            { return "boolean" }
func (n *Number) Type() string             { return "number" }
func (s *UnquotedString) Type() string     { return "unquoted string" }
func (s *DoubleQuotedString) Type() string { return "double quoted string" }
func (s *SingleQuotedString) Type() string { return "single quoted string" }
func (s *BlockString) Type() string        { return s.Tag + " block string" }
func (s *Substitution) Type() string       { return "substitution" }
func (a *Array) Type() string              { return "array" }
func (m *Map) Type() string                { return "map" }
func (k *Key) Type() string                { return "map key" }
func (k *KeyPath) Type() string            { return "key path" }
func (e *Edge) Type() string               { return "edge" }
func (i *EdgeIndex) Type() string          { return "edge index" }

func (c *Comment) GetRange() Range            { return c.Range }
func (c *BlockComment) GetRange() Range       { return c.Range }
func (n *Null) GetRange() Range               { return n.Range }
func (b *Boolean) GetRange() Range            { return b.Range }
func (n *Number) GetRange() Range             { return n.Range }
func (s *UnquotedString) GetRange() Range     { return s.Range }
func (s *DoubleQuotedString) GetRange() Range { return s.Range }
func (s *SingleQuotedString) GetRange() Range { return s.Range }
func (s *BlockString) GetRange() Range        { return s.Range }
func (s *Substitution) GetRange() Range       { return s.Range }
func (a *Array) GetRange() Range              { return a.Range }
func (m *Map) GetRange() Range                { return m.Range }
func (k *Key) GetRange() Range                { return k.Range }
func (k *KeyPath) GetRange() Range            { return k.Range }
func (e *Edge) GetRange() Range               { return e.Range }
func (i *EdgeIndex) GetRange() Range          { return i.Range }

func (c *Comment) mapNode()      {}
func (c *BlockComment) mapNode() {}
func (k *Key) mapNode()          {}
func (s *Substitution) mapNode() {}

func (c *Comment) arrayNode()            {}
func (c *BlockComment) arrayNode()       {}
func (n *Null) arrayNode()               {}
func (b *Boolean) arrayNode()            {}
func (n *Number) arrayNode()             {}
func (s *UnquotedString) arrayNode()     {}
func (s *DoubleQuotedString) arrayNode() {}
func (s *SingleQuotedString) arrayNode() {}
func (s *BlockString) arrayNode()        {}
func (s *Substitution) arrayNode()       {}
func (a *Array) arrayNode()              {}
func (m *Map) arrayNode()                {}

func (n *Null) value()               {}
func (b *Boolean) value()            {}
func (n *Number) value()             {}
func (s *UnquotedString) value()     {}
func (s *DoubleQuotedString) value() {}
func (s *SingleQuotedString) value() {}
func (s *BlockString) value()        {}
func (a *Array) value()              {}
func (m *Map) value()                {}

func (n *Null) scalar()               {}
func (b *Boolean) scalar()            {}
func (n *Number) scalar()             {}
func (s *UnquotedString) scalar()     {}
func (s *DoubleQuotedString) scalar() {}
func (s *SingleQuotedString) scalar() {}
func (s *BlockString) scalar()        {}

// TODO: mistake, move into parse.go
func (n *Null) ScalarString() string    { return n.Type() }
func (b *Boolean) ScalarString() string { return strconv.FormatBool(b.Value) }
func (n *Number) ScalarString() string  { return n.Raw }
func (s *UnquotedString) ScalarString() string {
	if len(s.Value) == 0 {
		return ""
	}
	if s.Value[0].String == nil {
		return ""
	}
	return *s.Value[0].String
}
func (s *DoubleQuotedString) ScalarString() string {
	if len(s.Value) == 0 {
		return ""
	}
	if s.Value[0].String == nil {
		return ""
	}
	return *s.Value[0].String
}
func (s *SingleQuotedString) ScalarString() string { return s.Value }
func (s *BlockString) ScalarString() string        { return s.Value }

func (s *UnquotedString) SetString(s2 string)     { s.Value = []InterpolationBox{{String: &s2}} }
func (s *DoubleQuotedString) SetString(s2 string) { s.Value = []InterpolationBox{{String: &s2}} }
func (s *SingleQuotedString) SetString(s2 string) { s.Raw = ""; s.Value = s2 }
func (s *BlockString) SetString(s2 string)        { s.Value = s2 }

func (s *UnquotedString) Copy() String     { tmp := *s; return &tmp }
func (s *DoubleQuotedString) Copy() String { tmp := *s; return &tmp }
func (s *SingleQuotedString) Copy() String { tmp := *s; return &tmp }
func (s *BlockString) Copy() String        { tmp := *s; return &tmp }

func (s *UnquotedString) _string()     {}
func (s *DoubleQuotedString) _string() {}
func (s *SingleQuotedString) _string() {}
func (s *BlockString) _string()        {}

type Comment struct {
	Range Range  `json:"range"`
	Value string `json:"value"`
}

type BlockComment struct {
	Range Range  `json:"range"`
	Value string `json:"value"`
}

type Null struct {
	Range Range `json:"range"`
}

type Boolean struct {
	Range Range `json:"range"`
	Value bool  `json:"value"`
}

type Number struct {
	Range Range    `json:"range"`
	Raw   string   `json:"raw"`
	Value *big.Rat `json:"value"`
}

type UnquotedString struct {
	Range Range              `json:"range"`
	Value []InterpolationBox `json:"value"`
}

func FlatUnquotedString(s string) *UnquotedString {
	return &UnquotedString{
		Value: []InterpolationBox{{String: &s}},
	}
}

type DoubleQuotedString struct {
	Range Range              `json:"range"`
	Value []InterpolationBox `json:"value"`
}

func FlatDoubleQuotedString(s string) *DoubleQuotedString {
	return &DoubleQuotedString{
		Value: []InterpolationBox{{String: &s}},
	}
}

type SingleQuotedString struct {
	Range Range  `json:"range"`
	Raw   string `json:"raw"`
	Value string `json:"value"`
}

type BlockString struct {
	Range Range `json:"range"`

	// Quote contains the pipe delimiter for the block string.
	// e.g. if 5 pipes were used to begin a block string, then Quote == "||||".
	// The tag is not included.
	Quote string `json:"quote"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

type Array struct {
	Range Range          `json:"range"`
	Nodes []ArrayNodeBox `json:"nodes"`
}

type Map struct {
	Range Range        `json:"range"`
	Nodes []MapNodeBox `json:"nodes"`
}

func (m *Map) InsertAfter(cursor, n MapNode) {
	afterIndex := len(m.Nodes) - 1

	for i, n := range m.Nodes {
		if n.Unbox() == cursor {
			afterIndex = i
		}
	}

	a := make([]MapNodeBox, 0, len(m.Nodes))
	a = append(a, m.Nodes[:afterIndex+1]...)
	a = append(a, MakeMapNodeBox(n))
	a = append(a, m.Nodes[afterIndex+1:]...)
	m.Nodes = a
}

func (m *Map) InsertBefore(cursor, n MapNode) {
	beforeIndex := len(m.Nodes)

	for i, n := range m.Nodes {
		if n.Unbox() == cursor {
			beforeIndex = i
		}
	}

	a := make([]MapNodeBox, 0, len(m.Nodes))
	a = append(a, m.Nodes[:beforeIndex]...)
	a = append(a, MakeMapNodeBox(n))
	a = append(a, m.Nodes[beforeIndex:]...)
	m.Nodes = a
}

func (m *Map) IsFileMap() bool {
	return m.Range.Start.Line == 0 && m.Range.Start.Column == 0
}

// TODO: require @ on import values for readability
type Key struct {
	Range Range `json:"range"`

	// Indicates this MapKey is an override selector.
	Ampersand bool `json:"ampersand,omitempty"`

	// At least one of Key and Edges will be set but all four can also be set.
	// The following are all valid MapKeys:
	// Key:
	//   x
	// Edges:
	//   x -> y
	// Edges and EdgeIndex:
	//   (x -> y)[*]
	// Edges and EdgeKey:
	//   (x -> y).label
	// Key and Edges:
	//   container.(x -> y)
	// Key, Edges and EdgeKey:
	//   container.(x -> y -> z).label
	// Key, Edges, EdgeIndex EdgeKey:
	//   container.(x -> y -> z)[4].label
	Key       *KeyPath   `json:"key,omitempty"`
	Edges     []*Edge    `json:"edges,omitempty"`
	EdgeIndex *EdgeIndex `json:"edge_index,omitempty"`
	EdgeKey   *KeyPath   `json:"edge_key,omitempty"`

	Primary ScalarBox `json:"primary,omitempty"`
	Value   ValueBox  `json:"value"`
}

// TODO there's more stuff to compare
func (mk1 *Key) Equals(mk2 *Key) bool {
	if (mk1.Key == nil) != (mk2.Key == nil) {
		return false
	}
	if (mk1.EdgeIndex == nil) != (mk2.EdgeIndex == nil) {
		return false
	}
	if (mk1.EdgeKey == nil) != (mk2.EdgeKey == nil) {
		return false
	}
	if len(mk1.Edges) != len(mk2.Edges) {
		return false
	}
	if (mk1.Value.Map == nil) != (mk2.Value.Map == nil) {
		return false
	}
	if (mk1.Value.Unbox() == nil) != (mk2.Value.Unbox() == nil) {
		return false
	}

	if mk1.Key != nil {
		if len(mk1.Key.Path) != len(mk2.Key.Path) {
			return false
		}
		for i, id := range mk1.Key.Path {
			if id.Unbox().ScalarString() != mk2.Key.Path[i].Unbox().ScalarString() {
				return false
			}
		}
	}

	if mk1.Value.Map != nil {
		if len(mk1.Value.Map.Nodes) != len(mk2.Value.Map.Nodes) {
			return false
		}
		for i := range mk1.Value.Map.Nodes {
			if !mk1.Value.Map.Nodes[i].MapKey.Equals(mk2.Value.Map.Nodes[i].MapKey) {
				return false
			}
		}
	}

	if mk1.Value.Unbox() != nil {
		if mk1.Value.ScalarBox().Unbox().ScalarString() != mk2.Value.ScalarBox().Unbox().ScalarString() {
			return false
		}
	}

	return true
}

func (mk *Key) SetScalar(scalar ScalarBox) {
	if mk.Value.Unbox() != nil && mk.Value.ScalarBox().Unbox() == nil {
		mk.Primary = scalar
	} else {
		mk.Value = MakeValueBox(scalar.Unbox())
	}
}

type KeyPath struct {
	Range Range        `json:"range"`
	Path  []*StringBox `json:"path"`
}

type Edge struct {
	Range Range `json:"range"`

	Src *KeyPath `json:"src"`
	// empty, < or *
	SrcArrow string `json:"src_arrow"`

	Dst *KeyPath `json:"dst"`
	// empty, > or *
	DstArrow string `json:"dst_arrow"`
}

type EdgeIndex struct {
	Range Range `json:"range"`

	// [n] or [*]
	Int  *int `json:"int"`
	Glob bool `json:"glob"`
}

type Substitution struct {
	Range Range `json:"range"`

	Spread bool         `json:"spread"`
	Path   []*StringBox `json:"path"`
}

// MapNodeBox is used to box MapNode for JSON persistence.
type MapNodeBox struct {
	Comment      *Comment      `json:"comment,omitempty"`
	BlockComment *BlockComment `json:"block_comment,omitempty"`
	Substitution *Substitution `json:"substitution,omitempty"`
	MapKey       *Key          `json:"map_key,omitempty"`
}

func MakeMapNodeBox(n MapNode) MapNodeBox {
	var box MapNodeBox
	switch n := n.(type) {
	case *Comment:
		box.Comment = n
	case *BlockComment:
		box.BlockComment = n
	case *Substitution:
		box.Substitution = n
	case *Key:
		box.MapKey = n
	}
	return box
}

func (mb MapNodeBox) Unbox() MapNode {
	switch {
	case mb.Comment != nil:
		return mb.Comment
	case mb.BlockComment != nil:
		return mb.BlockComment
	case mb.Substitution != nil:
		return mb.Substitution
	case mb.MapKey != nil:
		return mb.MapKey
	default:
		return nil
	}
}

// ArrayNodeBox is used to box ArrayNode for JSON persistence.
type ArrayNodeBox struct {
	Comment            *Comment            `json:"comment,omitempty"`
	BlockComment       *BlockComment       `json:"block_comment,omitempty"`
	Substitution       *Substitution       `json:"substitution,omitempty"`
	Null               *Null               `json:"null,omitempty"`
	Boolean            *Boolean            `json:"boolean,omitempty"`
	Number             *Number             `json:"number,omitempty"`
	UnquotedString     *UnquotedString     `json:"unquoted_string,omitempty"`
	DoubleQuotedString *DoubleQuotedString `json:"double_quoted_string,omitempty"`
	SingleQuotedString *SingleQuotedString `json:"single_quoted_string,omitempty"`
	BlockString        *BlockString        `json:"block_string,omitempty"`
	Array              *Array              `json:"array,omitempty"`
	Map                *Map                `json:"map,omitempty"`
}

func (ab ArrayNodeBox) Unbox() ArrayNode {
	switch {
	case ab.Comment != nil:
		return ab.Comment
	case ab.BlockComment != nil:
		return ab.BlockComment
	case ab.Substitution != nil:
		return ab.Substitution
	case ab.Null != nil:
		return ab.Null
	case ab.Boolean != nil:
		return ab.Boolean
	case ab.Number != nil:
		return ab.Number
	case ab.UnquotedString != nil:
		return ab.UnquotedString
	case ab.DoubleQuotedString != nil:
		return ab.DoubleQuotedString
	case ab.SingleQuotedString != nil:
		return ab.SingleQuotedString
	case ab.BlockString != nil:
		return ab.BlockString
	case ab.Array != nil:
		return ab.Array
	case ab.Map != nil:
		return ab.Map
	default:
		return nil
	}
}

// ValueBox is used to box Value for JSON persistence.
type ValueBox struct {
	Null               *Null               `json:"null,omitempty"`
	Boolean            *Boolean            `json:"boolean,omitempty"`
	Number             *Number             `json:"number,omitempty"`
	UnquotedString     *UnquotedString     `json:"unquoted_string,omitempty"`
	DoubleQuotedString *DoubleQuotedString `json:"double_quoted_string,omitempty"`
	SingleQuotedString *SingleQuotedString `json:"single_quoted_string,omitempty"`
	BlockString        *BlockString        `json:"block_string,omitempty"`
	Array              *Array              `json:"array,omitempty"`
	Map                *Map                `json:"map,omitempty"`
}

func (vb ValueBox) Unbox() Value {
	switch {
	case vb.Null != nil:
		return vb.Null
	case vb.Boolean != nil:
		return vb.Boolean
	case vb.Number != nil:
		return vb.Number
	case vb.UnquotedString != nil:
		return vb.UnquotedString
	case vb.DoubleQuotedString != nil:
		return vb.DoubleQuotedString
	case vb.SingleQuotedString != nil:
		return vb.SingleQuotedString
	case vb.BlockString != nil:
		return vb.BlockString
	case vb.Array != nil:
		return vb.Array
	case vb.Map != nil:
		return vb.Map
	default:
		return nil
	}
}

func MakeValueBox(v Value) ValueBox {
	var vb ValueBox
	switch v := v.(type) {
	case *Null:
		vb.Null = v
	case *Boolean:
		vb.Boolean = v
	case *Number:
		vb.Number = v
	case *UnquotedString:
		vb.UnquotedString = v
	case *DoubleQuotedString:
		vb.DoubleQuotedString = v
	case *SingleQuotedString:
		vb.SingleQuotedString = v
	case *BlockString:
		vb.BlockString = v
	case *Array:
		vb.Array = v
	case *Map:
		vb.Map = v
	}
	return vb
}

func (vb ValueBox) ScalarBox() ScalarBox {
	var sb ScalarBox
	sb.Null = vb.Null
	sb.Boolean = vb.Boolean
	sb.Number = vb.Number
	sb.UnquotedString = vb.UnquotedString
	sb.DoubleQuotedString = vb.DoubleQuotedString
	sb.SingleQuotedString = vb.SingleQuotedString
	sb.BlockString = vb.BlockString
	return sb
}

func (vb ValueBox) StringBox() *StringBox {
	var sb StringBox
	sb.UnquotedString = vb.UnquotedString
	sb.DoubleQuotedString = vb.DoubleQuotedString
	sb.SingleQuotedString = vb.SingleQuotedString
	sb.BlockString = vb.BlockString
	return &sb
}

// ScalarBox is used to box Scalar for JSON persistence.
// TODO: implement ScalarString()
type ScalarBox struct {
	Null               *Null               `json:"null,omitempty"`
	Boolean            *Boolean            `json:"boolean,omitempty"`
	Number             *Number             `json:"number,omitempty"`
	UnquotedString     *UnquotedString     `json:"unquoted_string,omitempty"`
	DoubleQuotedString *DoubleQuotedString `json:"double_quoted_string,omitempty"`
	SingleQuotedString *SingleQuotedString `json:"single_quoted_string,omitempty"`
	BlockString        *BlockString        `json:"block_string,omitempty"`
}

func (sb ScalarBox) Unbox() Scalar {
	switch {
	case sb.Null != nil:
		return sb.Null
	case sb.Boolean != nil:
		return sb.Boolean
	case sb.Number != nil:
		return sb.Number
	case sb.UnquotedString != nil:
		return sb.UnquotedString
	case sb.DoubleQuotedString != nil:
		return sb.DoubleQuotedString
	case sb.SingleQuotedString != nil:
		return sb.SingleQuotedString
	case sb.BlockString != nil:
		return sb.BlockString
	default:
		return nil
	}
}

// StringBox is used to box String for JSON persistence.
type StringBox struct {
	UnquotedString     *UnquotedString     `json:"unquoted_string,omitempty"`
	DoubleQuotedString *DoubleQuotedString `json:"double_quoted_string,omitempty"`
	SingleQuotedString *SingleQuotedString `json:"single_quoted_string,omitempty"`
	BlockString        *BlockString        `json:"block_string,omitempty"`
}

func (sb *StringBox) Unbox() String {
	switch {
	case sb.UnquotedString != nil:
		return sb.UnquotedString
	case sb.DoubleQuotedString != nil:
		return sb.DoubleQuotedString
	case sb.SingleQuotedString != nil:
		return sb.SingleQuotedString
	case sb.BlockString != nil:
		return sb.BlockString
	default:
		return nil
	}
}

// InterpolationBox is used to select between strings and substitutions in unquoted and
// double quoted strings. There is no corresponding interface to avoid unnecessary
// abstraction.
type InterpolationBox struct {
	String       *string       `json:"string,omitempty"`
	StringRaw    *string       `json:"raw_string,omitempty"`
	Substitution *Substitution `json:"substitution,omitempty"`
}

// & is only special if it begins a key.
// - is only special if followed by another - in a key.
// ' " and | are only special if they begin an unquoted key or value.
var UnquotedKeySpecials = string([]rune{'#', ';', '\n', '\\', '{', '}', '[', ']', '\'', '"', '|', ':', '.', '-', '<', '>', '*', '&', '(', ')'})
var UnquotedValueSpecials = string([]rune{'#', ';', '\n', '\\', '{', '}', '[', ']', '\'', '"', '|', '$'})

// RawString returns s in a AST String node that can format s in the most aesthetically
// pleasing way.
// TODO: Return StringBox
func RawString(s string, inKey bool) String {
	if s == "" {
		return FlatDoubleQuotedString(s)
	}

	if inKey {
		for i, r := range s {
			switch r {
			case '-':
				if i+1 < len(s) && s[i+1] != '-' {
					continue
				}
			case '&':
				if i > 0 {
					continue
				}
			}
			if strings.ContainsRune(UnquotedKeySpecials, r) {
				if !strings.ContainsRune(s, '"') {
					return FlatDoubleQuotedString(s)
				}
				if strings.ContainsRune(s, '\n') {
					return FlatDoubleQuotedString(s)
				}
				return &SingleQuotedString{Value: s}
			}
		}
	} else if s == "null" || strings.ContainsAny(s, UnquotedValueSpecials) {
		if !strings.ContainsRune(s, '"') && !strings.ContainsRune(s, '$') {
			return FlatDoubleQuotedString(s)
		}
		if strings.ContainsRune(s, '\n') {
			return FlatDoubleQuotedString(s)
		}
		return &SingleQuotedString{Value: s}
	}

	if hasSurroundingWhitespace(s) {
		return FlatDoubleQuotedString(s)
	}

	return FlatUnquotedString(s)
}

func hasSurroundingWhitespace(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	r2, _ := utf8.DecodeLastRuneInString(s)
	return unicode.IsSpace(r) || unicode.IsSpace(r2)
}
