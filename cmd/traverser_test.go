package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func TestWalkDir(t *testing.T) {
	// we will create a new directory, a couple files, then walk it and check it
	r := rand.Int63()
	testPath := fmt.Sprintf("./test_data/test_%d", r)
	rawD2 := fmt.Sprintf("%s/src/test%d.d2", testPath, r)
	rawMD := fmt.Sprintf("%s/src/test%d.md", testPath, r)

	err := os.MkdirAll(testPath+"/src", os.ModePerm)
	if err != nil {
		t.Fatalf("tried to create test src dir but could not: %v", err)
	}
	err = os.MkdirAll(testPath+"/build", os.ModePerm)
	if err != nil {
		t.Fatalf("tried to create test build dir but could not: %v", err)
	}

	file, err := os.Create(rawD2)
	if err != nil {
		t.Fatalf("tried to create test d2 file but could not: %v", err)
	}
	defer file.Close()
	_, err = file.Write([]byte(`a -> b`))
	if err != nil {
		t.Fatalf("tried to write test file but could not: %v", err)
	}
	file.Close()

	file, err = os.Create(rawMD)
	if err != nil {
		t.Fatalf("tried to create test md file but could not: %v", err)
	}
	defer file.Close()
	_, err = file.Write([]byte("---\ntitle: Test\n---\n\nHi!\n"))
	if err != nil {
		t.Fatalf("tried to write test file but could not: %v", err)
	}
	file.Close()

	err = execute(&CommandOptions{
		InputDirectory:               testPath + "/src",
		OutputDirectory:              testPath + "/build",
		PageTemplateFile:             "./default_templates/page.html",
		TagPageTemplateFile:          "./default_templates/tag.html",
		DiagramIndexPageTemplateFile: "./default_templates/diagram_index.html",
	})
	if err != nil {
		t.Fatalf("tried to walk but could not: %v", err)
	}

	os.RemoveAll(testPath)
}
