package main

import (
	"fmt"
	"github.com/swagger-generate/swagger2idl/converter"
	"github.com/swagger-generate/swagger2idl/generate"
	"github.com/swagger-generate/swagger2idl/parser"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Please provide the path to the OpenAPI file.")
	}

	openapiFile := os.Args[1]
	spec, err := parser.LoadOpenAPISpec(openapiFile)
	if err != nil {
		log.Fatalf("Failed to load OpenAPI file: %v", err)
	}

	fmt.Println("Successfully loaded OpenAPI specification!")

	yamlData, err := yaml.Marshal(spec.Paths.Map())
	if err != nil {
		fmt.Println("Error marshalling YAML:", err)
		return
	}

	// 创建或打开文件
	file, err := os.Create("internal/output.yaml")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close() // 确保文件在最后被关闭

	// 将 YAML 数据写入文件
	_, err = file.Write(yamlData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	// 创建 ProtoConverter 实例
	converter := converter.NewProtoConverter(strings.ReplaceAll(spec.Info.Title, " ", "_"))

	// 转换 OpenAPI 规范到 Proto
	err = converter.Convert(spec)
	if err != nil {
		panic(err)
	}

	// 生成 Proto 文件内容
	protoContent := generate.ConvertToProtoFile(converter.ProtoFile)

	// 输出 Proto 到文件
	protoFilename := "internal/output.proto"
	if len(os.Args) > 2 {
		protoFilename = os.Args[2]
	}

	f, err := os.Create(protoFilename)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	_, err = f.WriteString(protoContent)
}
