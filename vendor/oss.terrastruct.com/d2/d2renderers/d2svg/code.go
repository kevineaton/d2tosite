package d2svg

import (
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/svg"
)

// Copied private functions from chroma. Their public functions do too much (write the whole SVG document)
// https://github.com/alecthomas/chroma
// >>> BEGIN

var svgEscaper = strings.NewReplacer(
	`&`, "&amp;",
	`<`, "&lt;",
	`>`, "&gt;",
	`"`, "&quot;",
	` `, "&#160;",
	`	`, "&#160;&#160;&#160;&#160;",
)

func styleToSVG(style *chroma.Style) map[chroma.TokenType]string {
	converted := map[chroma.TokenType]string{}
	bg := style.Get(chroma.Background)
	// Convert the style.
	for t := range chroma.StandardTypes {
		entry := style.Get(t)
		if t != chroma.Background {
			entry = entry.Sub(bg)
		}
		if entry.IsZero() {
			continue
		}
		converted[t] = svg.StyleEntryToSVG(entry)
	}
	return converted
}

func styleAttr(styles map[chroma.TokenType]string, tt chroma.TokenType) string {
	if _, ok := styles[tt]; !ok {
		tt = tt.SubCategory()
		if _, ok := styles[tt]; !ok {
			tt = tt.Category()
			if _, ok := styles[tt]; !ok {
				return ""
			}
		}
	}
	return styles[tt]
}

// <<< END
