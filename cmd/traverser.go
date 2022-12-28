package cmd

import (
	"bytes"
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
	PageTemplate              string
	DiagramIndexPageTemplate  string
	TagPageTemplate           string
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
	leafTemplate := template.Must(template.ParseFiles(options.PageTemplate))
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

// buildTagPages builds each tag page that lists all of the pages that have a tag
func buildTagPages(options *CommandOptions) error {
	// we need to crate a tag page for each tag with links to each leaf with that tag
	leafTemplate := template.Must(template.ParseFiles(options.PageTemplate))
	tagTemplate := template.Must(template.ParseFiles(options.TagPageTemplate))
	// make sure the tags directory exists
	err := os.MkdirAll(options.OutputDirectory+"/tags/", os.ModePerm)
	if err != nil {
		return err
	}
	for tag, leaves := range site.SiteTags {
		tagFileName := strings.ReplaceAll(tag, " ", "_")
		// dir := options.OutputDirectory + "/tags/" + tag
		// os.MkdirAll(dir, os.ModePerm)
		var tagOutput bytes.Buffer
		err := tagTemplate.Execute(&tagOutput, map[string]interface{}{
			"Tag":    tag,
			"Leaves": leaves,
		})
		if err != nil {
			fmt.Printf("tag template error: %+v\n", err)
			continue
		}

		temp := d2s.LeafData{
			Title:    tag,
			Content:  template.HTML(tagOutput.String()),
			Links:    site.Links,
			SiteTags: site.SiteTags,
		}
		output, err := os.Create(options.OutputDirectory + "/tags/" + tagFileName + ".html")
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

// buildDiagramIndexPage builds the index of all the diagrams
func buildDiagramIndexPage(options *CommandOptions) error {
	leafTemplate := template.Must(template.ParseFiles(options.PageTemplate))
	diagramTemplate := template.Must(template.ParseFiles(options.DiagramIndexPageTemplate))
	var diagramOutput bytes.Buffer
	err := diagramTemplate.Execute(&diagramOutput, site)
	if err != nil {
		return err
	}
	temp := d2s.LeafData{
		Title:    "Index",
		Content:  template.HTML(diagramOutput.String()),
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
