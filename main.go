package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/hertz-contrib/swagger-generate/swagger2idl/converter"
	"github.com/hertz-contrib/swagger-generate/swagger2idl/generate"
	"github.com/hertz-contrib/swagger-generate/swagger2idl/parser"
	"github.com/urfave/cli/v2"
)

const defaultProtoFilename = "output.proto"
const defaultThriftFilename = "output.thrift"

var (
	outputType    string
	outputFile    string
	openapiOption bool
	apiOption     bool
	namingOption  bool
)

func main() {
	// Create a new CLI app
	app := &cli.App{
		Name:  "swagger2idl",
		Usage: "Convert OpenAPI specs to Protobuf or Thrift IDL",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "type",
				Aliases:     []string{"t"},
				Usage:       "Specify output type: 'proto' or 'thrift'. If not provided, inferred from output file extension.",
				Destination: &outputType,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "Specify output file path. If not provided, defaults to output.proto or output.thrift based on the output type.",
				Destination: &outputFile,
			},
			&cli.BoolFlag{
				Name:        "openapi",
				Aliases:     []string{"oa"},
				Usage:       "Include OpenAPI specific options in the output",
				Destination: &openapiOption,
			},
			&cli.BoolFlag{
				Name:        "api",
				Aliases:     []string{"a"},
				Usage:       "Include API specific options in the output",
				Destination: &apiOption,
			},
			&cli.BoolFlag{
				Name:        "naming",
				Aliases:     []string{"n"},
				Usage:       "use naming conventions for the output IDL file",
				Value:       true,
				Destination: &namingOption,
			},
		},
		Action: func(c *cli.Context) error {
			// Get remaining non-flag arguments (e.g., file paths)
			args := c.Args().Slice()

			// Ensure the OpenAPI file path is provided
			if len(args) < 1 {
				log.Fatal("Please provide the path to the OpenAPI file.")
			}

			openapiFile := args[0]

			// Automatically determine output type based on file extension if not provided
			if outputType == "" && outputFile != "" {
				ext := filepath.Ext(outputFile)
				switch ext {
				case ".proto":
					outputType = "proto"
				case ".thrift":
					outputType = "thrift"
				default:
					log.Fatalf("Cannot determine output type from file extension: %s. Use --type to specify explicitly.", ext)
				}
			}

			// If output file is not specified, use a default file based on outputType
			if outputFile == "" {
				if outputType == "proto" {
					outputFile = defaultProtoFilename
				} else if outputType == "thrift" {
					outputFile = defaultThriftFilename
				} else {
					log.Fatal("Output file must be specified if output type is not provided.")
				}
			}

			// Load the OpenAPI specification
			spec, err := parser.LoadOpenAPISpec(openapiFile)
			if err != nil {
				log.Fatalf("Failed to load OpenAPI file: %v", err)
			}

			// Initialize ConvertOption with command-line flag values
			converterOption := &converter.ConvertOption{
				OpenapiOption: openapiOption,
				ApiOption:     apiOption,
				NamingOption:  namingOption,
			}

			var idlContent string
			var file *os.File
			var errFile error

			switch outputType {
			case "proto":
				protoConv := converter.NewProtoConverter(spec, converterOption)

				if err = protoConv.Convert(); err != nil {
					log.Fatalf("Error during conversion: %v", err)
				}
				protoEngine := generate.NewProtoGenerate()

				idlContent, err = protoEngine.Generate(protoConv.GetIdl())

				file, errFile = os.Create(outputFile)
			case "thrift":
				thriftConv := converter.NewThriftConverter(spec, converterOption)

				if err = thriftConv.Convert(); err != nil {
					log.Fatalf("Error during conversion: %v", err)
				}
				thriftEngine := generate.NewThriftGenerate()

				idlContent, err = thriftEngine.Generate(thriftConv.GetIdl())

				file, errFile = os.Create(outputFile)
			default:
				log.Fatalf("Invalid output type: %s", outputType)
			}

			if errFile != nil {
				log.Fatalf("Failed to create file: %v", errFile)
			}
			defer func() {
				if err := file.Close(); err != nil {
					log.Printf("Error closing file: %v", err)
				}
			}()

			if _, err = file.WriteString(idlContent); err != nil {
				log.Fatalf("Error writing to file: %v", err)
			}

			return nil
		},
	}

	// Run the app
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
