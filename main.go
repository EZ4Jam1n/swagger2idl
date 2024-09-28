package main

import (
	"log"
	"os"
	"strings"

	"github.com/swagger-generate/swagger2idl/converter"
	"github.com/swagger-generate/swagger2idl/generate"
	"github.com/swagger-generate/swagger2idl/parser"
)

const defaultProtoFilename = "output.proto"

func main() {
	// Ensure the OpenAPI file path is provided as a command-line argument
	if len(os.Args) < 2 {
		log.Fatal("Please provide the path to the OpenAPI file.")
	}

	openapiFile := os.Args[1]

	// Load the OpenAPI specification
	spec, err := parser.LoadOpenAPISpec(openapiFile)
	if err != nil {
		log.Fatalf("Failed to load OpenAPI file: %v", err)
	}

	converter := converter.NewProtoConverter(strings.ReplaceAll(spec.Info.Title, " ", "_"))

	if err = converter.Convert(spec); err != nil {
		log.Fatalf("Error during conversion: %v", err)
	}

	protoContent := generate.ConvertToProtoFile(converter.ProtoFile)

	protoFilename := defaultProtoFilename
	if len(os.Args) > 2 {
		protoFilename = os.Args[2]
	}

	protoFile, err := os.Create(protoFilename)
	if err != nil {
		log.Fatalf("Failed to create Proto file: %v", err)
	}
	defer func() {
		if err := protoFile.Close(); err != nil {
			log.Printf("Error closing Proto file: %v", err)
		}
	}()

	if _, err = protoFile.WriteString(protoContent); err != nil {
		log.Fatalf("Error writing to Proto file: %v", err)
	}
}
