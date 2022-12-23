package parser_test

import (
	"html/template"
	"testing"

	parse "github.com/kevineaton/d2tosite/parser"
)

func TestProcessMarkdown(t *testing.T) {
	tests := []struct {
		Input               []byte
		Prefix              string
		ExpectedTitle       string
		ExpectedTags        []string
		ExpectedHTMLContent template.HTML
		ExpectAnError       bool
	}{
		{
			Input:               []byte{},
			Prefix:              "",
			ExpectedHTMLContent: "",
			ExpectedTitle:       "",
			ExpectedTags:        []string{},
			ExpectAnError:       true,
		},
		{
			Input:               []byte("# Header\n\nHi!"),
			Prefix:              "",
			ExpectedHTMLContent: "<h1>Header</h1>\n<p>Hi!</p>\n",
			ExpectedTitle:       "Header",
			ExpectedTags:        []string{},
			ExpectAnError:       false,
		},
		{
			Input:               []byte("# Header\n\n{{sample}}"),
			Prefix:              "",
			ExpectedHTMLContent: "<h1>Header</h1>\n<p><img src='sample.svg' alt='diagram' /></p>\n",
			ExpectedTitle:       "Header",
			ExpectedTags:        []string{},
			ExpectAnError:       false,
		},
		{
			Input:               []byte("# Header\n\n{{sample}}"),
			Prefix:              "/",
			ExpectedHTMLContent: "<h1>Header</h1>\n<p><img src='/sample.svg' alt='diagram' /></p>\n",
			ExpectedTitle:       "Header",
			ExpectedTags:        []string{},
			ExpectAnError:       false,
		},
		{
			Input:               []byte("# Header\n\n{{sample}}"),
			Prefix:              "test",
			ExpectedHTMLContent: "<h1>Header</h1>\n<p><img src='testsample.svg' alt='diagram' /></p>\n",
			ExpectedTitle:       "Header",
			ExpectedTags:        []string{},
			ExpectAnError:       false,
		},
		{
			Input:               []byte("# Header\n\n{{sample}}"),
			Prefix:              "/test/2/",
			ExpectedHTMLContent: "<h1>Header</h1>\n<p><img src='/test/2/sample.svg' alt='diagram' /></p>\n",
			ExpectedTitle:       "Header",
			ExpectedTags:        []string{},
			ExpectAnError:       false,
		},
		{
			Input:               []byte("---\ntitle: Meta!\ntags:\n  - one\n  - two\n---\n# Header\n\n{{sample}}"),
			Prefix:              "/test/2/",
			ExpectedHTMLContent: "<h1>Header</h1>\n<p><img src='/test/2/sample.svg' alt='diagram' /></p>\n",
			ExpectedTitle:       "Meta!",
			ExpectedTags:        []string{"one", "two"},
			ExpectAnError:       false,
		},
	}

	count := 0
	for _, tt := range tests {
		output, err := parse.ParseMD(tt.Input, tt.Prefix)
		if output.Content != tt.ExpectedHTMLContent {
			t.Errorf("expected HTML of %s but found %s", tt.ExpectedHTMLContent, output.Content)
		}
		if tt.ExpectedTitle != output.Title {
			t.Errorf("expected title '%s' but found '%s'", tt.ExpectedTitle, output.Title)
		}
		if len(tt.ExpectedTags) != len(output.Tags) {
			t.Errorf("expected %d tag(s) but found %d", len(tt.ExpectedTags), len(output.Tags))
		}
		if tt.ExpectAnError && err == nil {
			t.Errorf("test %d expected an error but it was nil", count)
		}
		count++
	}
	if count != len(tests) {
		t.Errorf("expected to run %d test but only ran %d", len(tests), count)
	}
}

func TestProcessD2(t *testing.T) {
	tests := []struct {
		Input              []byte
		ExpectedOutputSize int
		ParseOptions       *parse.ParseOptions
		ExpectAnError      bool
	}{
		{
			Input:         []byte{},
			ExpectAnError: true,
		},
		{
			Input:              []byte(`a -> b`),
			ExpectedOutputSize: 332304,
			ExpectAnError:      false,
		},
		{
			Input:              []byte(`a -> b`),
			ExpectedOutputSize: 332304,
			ExpectAnError:      false,
			ParseOptions: &parse.ParseOptions{
				D2Theme: 1,
			},
		},
	}

	count := 0
	for _, tt := range tests {
		output, err := parse.ParseD2(tt.Input, tt.ParseOptions)
		if len(output) != tt.ExpectedOutputSize {
			t.Errorf("expected output to be %d long but found %d", tt.ExpectedOutputSize, len(output))
		}
		if tt.ExpectAnError && err == nil {
			t.Errorf("test %d expected an error but it was nil", count)
		}
		count++
	}
	if count != len(tests) {
		t.Errorf("expected to run %d test but only ran %d", len(tests), count)
	}
}
