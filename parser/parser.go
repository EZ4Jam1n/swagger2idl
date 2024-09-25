package parser

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"os"
)

// LoadOpenAPISpec 解析 OpenAPI v3 文件
func LoadOpenAPISpec(filePath string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist", filePath)
	}

	// 从文件加载 OpenAPI 规范
	spec, err := loader.LoadFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %v", err)
	}

	// 校验规范
	if err := spec.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("failed to validate OpenAPI spec: %v", err)
	}

	return spec, nil
}
