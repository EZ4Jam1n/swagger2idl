package parser

import (
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

// LoadOpenAPISpec parses an OpenAPI spec from a file and returns it.
func LoadOpenAPISpec(filePath string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist", filePath)
	}

	spec, err := loader.LoadFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %v", err)
	}

	if err := spec.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("failed to validate OpenAPI spec: %v", err)
	}

	return spec, nil
}
