package parser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	"oss.terrastruct.com/d2/lib/textmeasure"
)

// these regexes are used to check for data within the markdown
var imageReplaceRegex = regexp.MustCompile(`{{(\w+)}}`)
var titleRegex = regexp.MustCompile(`<h1>(\w+)</h1>`)

// ParseOptions are options relevants specifically to parsing, usually
// filled in automatically from the CommandOptions if run from the binary
type ParseOptions struct {
	D2Theme      int64
	D2OutputType string // currently, this is ignored as the d2 lib only really supports svg outputs
}

// LeafData holds the data for a leaf that will then be used to build the site
type LeafData struct {
	Title    string
	FileName string
	Tags     []string
	SiteTags map[string][]LeafData // needed for the nav
	Links    []LeafData            // needed for the nav
	Diagrams []string              // needed for the index
	Content  template.HTML         // used for converting to an html template
	Summary  string                // used for search displays, found in the meta
}

// ParseMD takes a series of bytes, such as from a file, and parses the MD into HTML, with meta data set
// in the LeafData return
func ParseMD(content []byte, prefix string) (*LeafData, error) {
	data := &LeafData{}

	if len(content) == 0 {
		return data, errors.New("invalid markdown content")
	}

	markdown := goldmark.New(goldmark.WithExtensions(meta.Meta))
	var buf bytes.Buffer
	pctx := parser.NewContext()
	err := markdown.Convert(content, &buf, parser.WithContext(pctx))
	if err != nil {
		return data, err
	}
	meta := meta.Get(pctx)
	output := buf.Bytes()

	tags := []string{}
	if t, tOK := meta["tags"]; tOK {
		if converted, cOK := t.([]interface{}); cOK {
			for i := range converted {
				tags = append(tags, fmt.Sprintf("%v", converted[i]))
			}
		}
	}

	title := ""
	if t, tOK := meta["title"]; tOK {
		if converted, cOK := t.(string); cOK {
			title = converted
		}
	}

	summary := ""
	if t, tOK := meta["summary"]; tOK {
		if converted, cOK := t.(string); cOK {
			summary = converted
		}
	}

	if summary == "" {
		summary = fmt.Sprintf("%s's content and information", title)
	}

	// replace images
	matches := imageReplaceRegex.FindAll(output, -1)
	for i := range matches {
		diagram := strings.ReplaceAll(string(matches[i]), "{{", prefix)
		diagram = strings.ReplaceAll(diagram, "}}", ".svg")
		data.Diagrams = append(data.Diagrams, diagram)
	}
	output = imageReplaceRegex.ReplaceAll(output, []byte(fmt.Sprintf("<img src='%s$1.svg' class='diagram-svg' alt='diagram' />", prefix)))

	// we want to make sure our pages are generally correct, so we need to split here
	// first, if there's no title, we need to see if it's in the markdown by default
	// then, if the title WAS provided, we want to add it to the top of the content IF
	// there isn't one already
	if title == "" {
		titleBytes := titleRegex.Find(output)
		if len(titleBytes) > 0 {
			title = strings.Replace(string(titleBytes), "<h1>", "", -1)
			title = strings.Replace(title, "</h1>", "", -1)
		}
		// if it's still blank, it's unknown, and the caller can handle that
	} else {
		// title is known, we need to see if we need to add it; since this is pretty
		// straight forward, we only need to see if the <h1> tag exists. If it does,
		// we can assume that the title is in the content so don't worry about adding it
		if !strings.Contains(string(output), "<h1>") {
			output = append([]byte(fmt.Sprintf("<h1>%s</h1>\n", title)), output...)
		}
	}

	data.Content = template.HTML(output)
	data.Title = title
	data.Tags = tags
	data.Summary = summary
	return data, nil
}

// ParseD2 takes in the bytes, such as from a file or a stream, and processes it through
// the D2 library for output
func ParseD2(input []byte, options *ParseOptions) ([]byte, error) {
	bytes := []byte{}
	if len(input) == 0 {
		return bytes, errors.New("invalid input")
	}
	if options == nil {
		options = &ParseOptions{
			D2Theme:      1,
			D2OutputType: "svg",
		}
	}
	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return bytes, err
	}
	diagram, _, err := d2lib.Compile(context.Background(), string(input), &d2lib.CompileOptions{
		Layout:  d2dagrelayout.Layout,
		Ruler:   ruler,
		ThemeID: options.D2Theme,
	})
	if err != nil {
		return bytes, err
	}

	out, err := d2svg.Render(diagram, d2svg.DEFAULT_PADDING)

	if err != nil {
		return bytes, err
	}

	return out, nil
}
