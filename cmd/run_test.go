package cmd

import (
	"testing"
)

func TestValidateOptions(t *testing.T) {
	input := &CommandOptions{}

	err := validateOptions(input)
	if err == nil {
		t.Error("expected error to not be nil since input directory doesn't exist")
	}
	input.InputDirectory = "."
	err = validateOptions(input)
	if err == nil {
		t.Error("expected error to not be nil since index template doesn't exist")
	}
	err = validateOptions(input)
	if err == nil {
		t.Error("expected error to not be nil since page template doesn't exist")
	}
	input.PageTemplateFile = "./default_templates/page.html"
	err = validateOptions(input)
	if err != nil {
		t.Errorf("expected error to be nil: %+v", err)
	}
}

func TestRun(t *testing.T) {
	err := Run()
	if err == nil {
		t.Errorf("should run even if incorrect flags")
	}
}
