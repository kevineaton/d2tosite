package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func TestValidateOptions(t *testing.T) {
	input := &CommandOptions{}

	err := validateOptions(input)
	if err == nil {
		t.Error("expected error to not be nil since input directory doesn't exist")
	}
	input.PageTemplateFile = "./default_templates/page.html"
	err = validateOptions(input)
	if err == nil {
		t.Errorf("expected error since the page template does not exist: %+v", input)
	}
}

func TestRun(t *testing.T) {
	err := Run()
	if err == nil {
		t.Errorf("should run even if incorrect flags")
	}
}

func TestParseConfigFile(t *testing.T) {
	r := rand.Int63()
	testDir := fmt.Sprintf("./test_data/%d", r)
	err := os.MkdirAll(testDir, os.ModePerm)
	if err != nil {
		t.Fatalf("could not create filesystem: %v", err)
	}
	defer os.RemoveAll(testDir)
	jsonFilePath := fmt.Sprintf("%s/%d_config.json", testDir, r)
	yamlFilePath := fmt.Sprintf("%s/%d_config.yaml", testDir, r)
	unsupportedFilePath := fmt.Sprintf("%s/%d_config.txt", testDir, r)

	// write the files
	jsonFile, err := os.Create(jsonFilePath)
	if err != nil {
		t.Errorf("could not create json file: %v", err)
	}
	jsonFile.Write([]byte(`{
		"d2_theme": 3
	}`))
	jsonFile.Close()

	yamlFile, err := os.Create(yamlFilePath)
	if err != nil {
		t.Errorf("could not create yaml file: %v", err)
	}
	yamlFile.Write([]byte(`d2_theme: 4`))
	yamlFile.Close()

	unsupportedFile, err := os.Create(unsupportedFilePath)
	if err != nil {
		t.Errorf("could not create unsupported file: %v", err)
	}
	unsupportedFile.Write([]byte(`d2_theme = 2`))
	unsupportedFile.Close()

	// now, try to parse each one and make sure the overrides happen
	options := &CommandOptions{
		PageTemplateFile: "test",
		ConfigFile:       jsonFilePath,
	}
	err = parseConfiguration(options)
	if err != nil {
		t.Errorf("error parsing %s: %v", options.ConfigFile, err)
	}
	if options.D2Theme != 3 {
		t.Errorf("expected theme to be 3 but was: %d", options.D2Theme)
	}
	if options.PageTemplateFile != "test" {
		t.Errorf("expected template file to be the same but was changed: %s", options.PageTemplateFile)
	}

	// reset and read yaml
	options.D2Theme = 0
	options.ConfigFile = yamlFilePath
	err = parseConfiguration(options)
	if err != nil {
		t.Errorf("error parsing %s: %v", options.ConfigFile, err)
	}
	if options.D2Theme != 4 {
		t.Errorf("expected theme to be 4 but was: %d", options.D2Theme)
	}
	if options.PageTemplateFile != "test" {
		t.Errorf("expected template file to be the same but was changed: %s", options.PageTemplateFile)
	}

	// reset and error on unsupported
	options.D2Theme = 1
	options.ConfigFile = unsupportedFilePath
	err = parseConfiguration(options)
	if err == nil {
		t.Errorf("expected error parsing %s: %v", options.ConfigFile, err)
	}
	if options.D2Theme != 1 {
		t.Errorf("expected theme to be 1 but was: %d", options.D2Theme)
	}
	if options.PageTemplateFile != "test" {
		t.Errorf("expected template file to be the same but was changed: %s", options.PageTemplateFile)
	}

}
