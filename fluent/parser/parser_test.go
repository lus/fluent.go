package parser

import (
	"encoding/json"
	"github.com/lus/fluent.go/fluent/parser/ast"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFixtures(t *testing.T) {
	// Collect the file names (without extension) from the fixtures in the '../../test/fixtures' directory
	var fileNames []string
	filepath.Walk(filepath.Join("../../test", "fixtures"), func(path string, info fs.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".ftl") {
			fileNames = append(fileNames, strings.TrimSuffix(info.Name(), ".ftl"))
		}
		return nil
	})

	for _, fileName := range fileNames {
		// Read the FTL input of the fixture
		input, err := ioutil.ReadFile(filepath.Join("../../test", "fixtures", fileName+".ftl"))
		if err != nil {
			t.Fatal(err)
		}

		// Parse the input
		resource, _ := New(string(input)).Parse()
		if resource == nil {
			t.Fatal("parsed resource is nil")
		}
		for _, entry := range resource.Body {
			// Remove junk annotations as these are excluded in the fixtures
			if junk, ok := entry.(*ast.Junk); ok {
				junk.Annotations = []string{}
			}
		}

		// Marshal the parsed AST into JSON and unmarshal it into a map
		// This is simply done to not include any 3rd party dependencies that directly perform struct -> map marshalling
		resourceJson, err := json.Marshal(resource)
		if err != nil {
			t.Fatal(err)
		}
		resourceMap := make(map[string]interface{})
		if err := json.Unmarshal(resourceJson, &resourceMap); err != nil {
			t.Fatal(err)
		}

		// Read the expected AST from the fixture and marshal it into a map too
		expectedOutputJson, err := ioutil.ReadFile(filepath.Join("../../test", "fixtures", fileName+".json"))
		if err != nil {
			t.Fatal(err)
		}
		expectedOutputMap := make(map[string]interface{})
		if err := json.Unmarshal(expectedOutputJson, &expectedOutputMap); err != nil {
			t.Fatal(err)
		}

		// Both maps have to match in order to pass the test
		matches := reflect.DeepEqual(resourceMap, expectedOutputMap)
		if !matches {
			t.Fatal("produced output does not match the expectation")
		}
	}
}
