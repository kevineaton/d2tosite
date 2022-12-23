package cmd

import (
	"os"
	"path/filepath"
	"strings"

	d2s "github.com/kevineaton/d2tosite/parser"
)

// handleMD takes a string path to an MD file and then reads it and hands it off to the library
func handleMD(inputFile, prefix string) (*d2s.LeafData, error) {
	// process the md
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, err
	}

	data, err := d2s.ParseMD(content, prefix)
	if err != nil {
		return data, err
	}

	if data.Title == "" {
		data.Title = strings.TrimRight(filepath.Base(inputFile), filepath.Ext(inputFile))
	}
	data.FileName = prefix + strings.Replace(filepath.Base(inputFile), ".md", ".html", -1)
	return data, err
}

// handleD2 takes a string for an input file and processes it
func handleD2(inputFile string, outputFile string, options *d2s.ParseOptions) error {
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}

	out, err := d2s.ParseD2(content, options)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(outputFile), out, 0600)
	if err != nil {
		return err
	}
	return nil
}

func handleOther(inputFile string, outputFile string) error {
	b, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}
	err = os.WriteFile(outputFile, b, os.ModePerm)
	return err
}
