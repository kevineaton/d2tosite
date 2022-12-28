package cmd

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	d2s "github.com/kevineaton/d2tosite/parser"
)

// CommandOptions holds all of the options to pass in to the processors
type CommandOptions struct {
	D2Theme                   int64
	D2OutputType              string // currently, this is ignored as the d2 lib only really supports svg outputs
	InputDirectory            string
	OutputDirectory           string
	LeafTemplate              string
	CleanOutputDirectoryFirst bool
	ContinueOnCompileErrors   bool
}

// SiteData is the main store of the site data
type SiteData struct {
	Title       string
	Content     template.HTML
	Links       []d2s.LeafData
	Tags        []string
	SiteTags    map[string][]d2s.LeafData
	AllDiagrams map[string]*d2s.LeafData
}

var site *SiteData

func setupSite() {
	site = &SiteData{}
	site.Title = ""
	site.Content = ""
	site.Links = []d2s.LeafData{}
	site.Tags = []string{}
	site.SiteTags = map[string][]d2s.LeafData{}
	site.AllDiagrams = map[string]*d2s.LeafData{}
}

var traverseErrors = []error{}

// walkInputDirectory walks the input directory to generate the desired site
func walkInputDirectory(options *CommandOptions) error {
	if site == nil {
		setupSite()
	}
	inputPath := options.InputDirectory
	outputPath := options.OutputDirectory
	parseOptions := &d2s.ParseOptions{
		D2Theme:      options.D2Theme,
		D2OutputType: options.D2OutputType,
	}

	fsys := os.DirFS(inputPath)
	// first, make sure the output directory is created
	err := os.MkdirAll(options.OutputDirectory, os.ModePerm)
	if err != nil {
		return err
	}
	fs.WalkDir(fsys, ".", func(path string, d os.DirEntry, walkErr error) error {

		// errors are handled a bit differently here; since we want to continue traversing,
		// we will compile all errors into the slice of errors and report on them after

		if path == "." {
			// we don't need this, so we skip
			return nil
		}

		// if it's not root, let's make sure that the output path is there
		if d.IsDir() {
			output := filepath.Join(outputPath, path)
			err := os.MkdirAll(output, os.ModePerm)
			if err != nil {
				traverseErrors = append(traverseErrors, fmt.Errorf("%s: %+v", output, err))
			}
			return nil
		}

		// it's a file, so let's set up the correct targets
		inputFile := filepath.Join(inputPath, path)
		outputFile := filepath.Join(outputPath, path)

		switch filepath.Ext(path) {
		case ".d2":
			// if it's a d2 diagram, hand it off for compilation
			outputFile = strings.TrimSuffix(outputFile, filepath.Ext(path))
			outputFile += "." + options.D2OutputType
			err := handleD2(inputFile, outputFile, parseOptions)
			if err != nil {
				traverseErrors = append(traverseErrors, fmt.Errorf("%s: %+v", outputFile, err))
			}
			return nil
		case ".md":
			// if it's markdown, process it and prepare it for conversion
			prefix := string(os.PathSeparator) + strings.TrimRight(path, filepath.Base(path))
			leaf, err := handleMD(inputFile, prefix)
			if err != nil {
				traverseErrors = append(traverseErrors, fmt.Errorf("%s: %+v", outputFile, err))
			}
			site.Links = append(site.Links, *leaf)
			for _, tag := range leaf.Tags {
				site.SiteTags[tag] = append(site.SiteTags[tag], *leaf)
			}

			for _, diagram := range leaf.Diagrams {
				site.AllDiagrams[diagram] = leaf
			}
		default:
			// we just want to copy the file
			err := handleOther(inputFile, outputFile)
			if err != nil {
				traverseErrors = append(traverseErrors, fmt.Errorf("%s: %+v", outputFile, err))
			}
		}

		return nil
	})
	return nil
}

// processTemplates handles taking the walked file system and changing
// the site into a serials of templates
func processTemplates(options *CommandOptions) error {
	leafTemplate := template.Must(template.ParseFiles(options.LeafTemplate))
	for i := range site.Links {
		if site.Links[i].Title == "" {
			continue
		}
		output, err := os.Create(options.OutputDirectory + "/" + site.Links[i].FileName)
		if err != nil {
			return err
		}
		defer output.Close()
		site.Links[i].Links = site.Links
		site.Links[i].SiteTags = site.SiteTags
		err = leafTemplate.Execute(output, site.Links[i])
		if err != nil {
			return err
		}
	}
	// now build a default Search page
	output, err := os.Create(options.OutputDirectory + "/search.html")
	if err != nil {
		fmt.Printf("\nHere: %+v\n", err)
		return err
	}
	defer output.Close()
	searchPage := &d2s.LeafData{
		Title:    "Search",
		Content:  "<h1>Search Results</h1>",
		Links:    site.Links,
		SiteTags: site.SiteTags,
	}
	err = leafTemplate.Execute(output, searchPage)
	return err
}

func buildTagPages(options *CommandOptions) error {
	// we need to crate a tag page for each tag with links to each leaf with that tag
	leafTemplate := template.Must(template.ParseFiles(options.LeafTemplate))
	for tag, leaves := range site.SiteTags {
		dir := options.OutputDirectory + "/tags/" + tag
		os.MkdirAll(dir, os.ModePerm)
		sb := strings.Builder{}
		sb.WriteString(fmt.Sprintf("<strong>Tag %s</strong><ul>", tag))
		for i := range leaves {
			sb.WriteString(fmt.Sprintf("<li><a href='%s'>%s</a></li>", leaves[i].FileName, leaves[i].Title))
		}
		sb.WriteString("</ul>")
		html := sb.String()
		temp := d2s.LeafData{
			Title:    tag,
			Content:  template.HTML(html),
			Links:    site.Links,
			SiteTags: site.SiteTags,
		}
		output, err := os.Create(dir + "/index.html")
		if err != nil {
			return err
		}
		defer output.Close()
		err = leafTemplate.Execute(output, temp)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildIndexPage(options *CommandOptions) error {
	leafTemplate := template.Must(template.ParseFiles(options.LeafTemplate))
	sb := strings.Builder{}
	sb.WriteString("<h1>Diagram Index</h1>")
	if len(site.AllDiagrams) == 0 {
		sb.WriteString("<p>No diagrams produced</p>")
	}
	for diagram, leaf := range site.AllDiagrams {
		sb.WriteString(fmt.Sprintf(`
			<div class="diagram-index-container">
				<div class="row">
					<div class="col-3">
					<a href="%s" target="_%s" class="diagram-index-link diagram-index-link-title">%s</a>
					</div>
					<div class="col-7">
						<a href="%s" target="_%s" class="diagram-index-link diagram-index-link-path">%s</a>
					</div>
					<div class="col-2">
					</div>
				</div>
				<div class="row">
					<div class="col-10 offset-1">
						<strong>Summary</strong><br />
						%s
					</div>
				</div>
			</div>`,
			diagram,
			diagram,
			leaf.Title,
			diagram,
			diagram,
			diagram,
			leaf.Summary))
	}
	html := sb.String()
	temp := d2s.LeafData{
		Title:    "Index",
		Content:  template.HTML(html),
		Links:    site.Links,
		SiteTags: site.SiteTags,
	}
	output, err := os.Create(options.OutputDirectory + "/diagram_index.html")
	if err != nil {
		return err
	}
	defer output.Close()
	err = leafTemplate.Execute(output, temp)
	if err != nil {
		return err
	}
	return nil
}
