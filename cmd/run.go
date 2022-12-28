package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"os"

	"github.com/urfave/cli"
)

//go:embed default_templates/page.html
var pageTemplateEmbedString string

//go:embed default_templates/tag.html
var tagTemplateEmbedString string

//go:embed default_templates/diagram_index.html
var diagramIndexTemplateEmbedString string

// Run is the main entrypoint for the binary. It takes various options and then works through the process
func Run() error {
	options := &CommandOptions{}
	app := &cli.App{
		Name:        "d2tosite",
		Description: "A simple CLI that traverses a directory and generates a basic HTML site from Markdown and D2 files",
		Usage:       "Use it",
		Flags: []cli.Flag{
			&cli.Int64Flag{
				Name:        "d2-theme",
				Value:       1,
				Usage:       "the D2 theme ID to use",
				Destination: &options.D2Theme,
			},
			&cli.StringFlag{
				Name:        "input-directory",
				Value:       "./src",
				Usage:       "the directory to read from and walk to build the site",
				Destination: &options.InputDirectory,
			},
			&cli.StringFlag{
				Name:        "output-directory",
				Value:       "./build",
				Usage:       "the output directory to publish the site to",
				Destination: &options.OutputDirectory,
			},
			&cli.StringFlag{
				Name:        "page-template",
				Value:       "",
				Usage:       "the template to use for each page; if not provided, it will used the embedded template file at compile time",
				Destination: &options.PageTemplateFile,
			},
			&cli.StringFlag{
				Name:        "index-template",
				Value:       "",
				Usage:       "the template to use for the content of the diagram index; if not provided, it will used the embedded template file at compile time",
				Destination: &options.PageTemplateFile,
			},
			&cli.StringFlag{
				Name:        "tag-template",
				Value:       "",
				Usage:       "the template to use for each tag page content; if not provided, it will used the embedded template file at compile time",
				Destination: &options.PageTemplateFile,
			},
			&cli.BoolFlag{
				Name:        "clean",
				Usage:       "if true, removes the target build directory prior to build",
				Destination: &options.CleanOutputDirectoryFirst,
			},
			&cli.BoolFlag{
				Name:        "continue-errors",
				Usage:       "if true, continues to build site after parsing and compiling errors are found",
				Destination: &options.ContinueOnCompileErrors,
			},
		},
		Action: func(context *cli.Context) error {
			// check the arguments; if there's 2, then we override what is
			// in the options
			argInput := context.Args().Get(0)
			argOutput := context.Args().Get(1)
			if argInput != "" && argOutput != "" {
				options.InputDirectory = argInput
				options.OutputDirectory = argOutput
			}
			err := execute(options)
			return err
		},
	}
	err := app.Run(os.Args)
	return err
}

// execute actually executes based upon the options
func execute(options *CommandOptions) error {
	err := validateOptions(options)
	if err != nil {
		return err
	}
	if options.CleanOutputDirectoryFirst {
		err = os.RemoveAll(options.OutputDirectory)
		if err != nil {
			return err
		}
	}
	err = walkInputDirectory(options)
	if err != nil { // this will almost always be nil
		return err
	}
	if len(traverseErrors) != 0 {
		for i := range traverseErrors {
			fmt.Printf("error: %+v\n", traverseErrors[i])
		}
		if options.ContinueOnCompileErrors {
			fmt.Printf("options set to continue on errors, continuing...\n")
		} else {
			return errors.New("errors encountered, stopping")
		}
	}
	err = processTemplates(options)
	if err != nil {
		return err
	}
	err = buildTagPages(options)
	if err != nil {
		return err
	}
	err = buildDiagramIndexPage(options)
	return err
}

// validateOptions validates the options prior to running
func validateOptions(options *CommandOptions) error {
	var err error
	if options == nil {
		options = &CommandOptions{}
	}
	if options.D2Theme == 0 {
		options.D2Theme = 8
	}

	// now we want to validate the templates; if one isn't provided
	// we will use the embedded ones. Effectively, check if the template exists
	// and if either it doesn't or a filename wasn't provided, fall back to the
	// embedded ones

	if options.PageTemplateFile != "" {
		foundTemplate, err := template.ParseFiles(options.PageTemplateFile)
		if err != nil {
			// we couldn't parse it, so show an error and load the template
			fmt.Printf("error: could not find page template: %s\n", options.PageTemplateFile)
			// if we close on errors, close
			if !options.ContinueOnCompileErrors {
				return err
			}
		} else {
			options.PageTemplate = foundTemplate
		}
	}
	// check if the template is nil from either not being provided a valid file OR the input was blank
	if options.PageTemplate == nil {
		foundTemplate, err := template.New("pageTemplate").Parse(pageTemplateEmbedString)
		if err != nil {
			return err
		}
		options.PageTemplate = foundTemplate
	}

	// repeat for tag template
	if options.TagPageTemplateFile != "" {
		foundTemplate, err := template.ParseFiles(options.TagPageTemplateFile)
		if err != nil {
			// we couldn't parse it, so show an error and load the template
			fmt.Printf("error: could not find tag template: %s\n", options.TagPageTemplateFile)
			// if we close on errors, close
			if !options.ContinueOnCompileErrors {
				return err
			}
		} else {
			options.TagPageTemplate = foundTemplate
		}
	}
	// check if the template is nil from either not being provided a valid file OR the input was blank
	if options.TagPageTemplate == nil {
		foundTemplate, err := template.New("tagTemplate").Parse(tagTemplateEmbedString)
		if err != nil {
			return err
		}
		options.TagPageTemplate = foundTemplate
	}

	// again for diagram index
	if options.DiagramIndexPageTemplateFile != "" {
		foundTemplate, err := template.ParseFiles(options.DiagramIndexPageTemplateFile)
		if err != nil {
			// we couldn't parse it, so show an error and load the template
			fmt.Printf("error: could not find diagram index template: %s\n", options.DiagramIndexPageTemplateFile)
			// if we close on errors, close
			if !options.ContinueOnCompileErrors {
				return err
			}
		} else {
			options.DiagramIndexPageTemplate = foundTemplate
		}
	}
	// check if the template is nil from either not being provided a valid file OR the input was blank
	if options.DiagramIndexPageTemplate == nil {
		foundTemplate, err := template.New("diagramIndexTemplate").Parse(diagramIndexTemplateEmbedString)
		if err != nil {
			return err
		}
		options.DiagramIndexPageTemplate = foundTemplate
	}

	// now we need to stat the input
	if _, err := os.Stat(options.InputDirectory); os.IsNotExist(err) {
		return fmt.Errorf("input directory %s does not exist, terminating", options.InputDirectory)
	}

	return err
}
