package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli"
)

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
				Name:        "d2-output-type",
				Value:       "svg",
				Usage:       "the output type for the d2 compiler; can only be svg at this time and is otherwise ignored",
				Destination: &options.D2OutputType,
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
				Name:        "leaf-template",
				Value:       "./default_templates/leaf.html",
				Usage:       "the template to use for each leaf; if not provided it will use the included default",
				Destination: &options.LeafTemplate,
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
			err := validateOptions(options)
			if err != nil {
				return err
			}
			err = execute(options)
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
	err = walkInputDirectory(options)
	if err != nil { // this will almost always be nil
		return err
	}
	if len(traverseErrors) != 0 {
		for i := range traverseErrors {
			fmt.Printf("error: %+v\n", traverseErrors[i])
		}
		return errors.New("errors encountered, stopping")
	}
	err = processTemplates(options)
	if err != nil {
		return err
	}
	err = buildTagPages(options)
	if err != nil {
		return err
	}
	err = buildIndexPage(options)
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
	options.D2OutputType = "svg"
	if options.LeafTemplate == "" {
		options.LeafTemplate = "./cmd/default_templates/leaf.html"
	}

	// now we need to stat the templates and input
	if _, err := os.Stat(options.InputDirectory); os.IsNotExist(err) {
		return fmt.Errorf("input directory %s does not exist, terminating", options.InputDirectory)
	}
	if _, err := os.Stat(options.LeafTemplate); os.IsNotExist(err) {
		return fmt.Errorf("leaf template %s does not exist, terminating", options.LeafTemplate)
	}

	return err
}
