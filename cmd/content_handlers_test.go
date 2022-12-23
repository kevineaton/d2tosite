package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/kevineaton/d2tosite/parser"
)

func TestD2Handler(t *testing.T) {
	// for this, we write a file, read it, then delete it
	r := rand.Int63()
	inputFileName := fmt.Sprintf("./test_data/test%d.d2", r)
	outputFileName := fmt.Sprintf("./test_data/test%d.svg", r)
	file, err := os.Create(inputFileName)
	if err != nil {
		t.Fatalf("tried to create test file but could not: %v", err)
	}
	defer file.Close()
	_, err = file.Write([]byte(`a -> b`))
	if err != nil {
		t.Fatalf("tried to write test file but could not: %v", err)
	}
	file.Close()

	// process it
	err = handleD2(inputFileName, outputFileName, &parser.ParseOptions{})
	if err != nil {
		t.Fatalf("tried to handle test file but could not: %v", err)
	}

	os.Remove(inputFileName)
	os.Remove(outputFileName)
}

func TestMDHandler(t *testing.T) {
	// for this, we write a file then delete it
	r := rand.Int63()
	inputFileName := fmt.Sprintf("./test_data/test%d.md", r)
	outputFileName := fmt.Sprintf("./test_data/test%d.html", r)
	file, err := os.Create(inputFileName)
	if err != nil {
		t.Fatalf("tried to create test file but could not: %v", err)
	}
	defer file.Close()
	_, err = file.Write([]byte("# Heading\n\nHi!\n"))
	if err != nil {
		t.Fatalf("tried to write test file but could not: %v", err)
	}
	file.Close()

	// process it
	data, err := handleMD(inputFileName, "")
	if err != nil {
		t.Fatalf("tried to handle test file but could not: %v", err)
	}
	if data.Title != "Heading" {
		t.Errorf("expected title of 'Heading' but found '%s'", data.Title)
	}
	if data.Content != "<h1>Heading</h1>\n<p>Hi!</p>\n" {
		t.Errorf("expected content of '<h1>Heading</h1>\n<p>Hi!</p>\n' but found '%s'", data.Content)
	}

	os.Remove(inputFileName)
	os.Remove(outputFileName)
}
