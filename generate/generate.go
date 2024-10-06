/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package generate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hertz-contrib/swagger-generate/swagger2idl/protobuf"
	"github.com/hertz-contrib/swagger-generate/swagger2idl/utils"
)

// Encoder is used to handle the encoding context
type Encoder struct {
	dst *strings.Builder // The target for output
}

// NewEncoder creates a new Encoder instance
func NewEncoder() *Encoder {
	return &Encoder{dst: &strings.Builder{}}
}

// ConvertToProtoFile converts the ProtoFile structure into Proto file content
func (e *Encoder) ConvertToProtoFile(protoFile *protobuf.ProtoFile) string {

	e.dst.WriteString("syntax = \"proto3\";\n\n")
	e.dst.WriteString(fmt.Sprintf("package %s;\n\n", protoFile.PackageName))

	// Generate imports
	for _, importFile := range protoFile.Imports {
		e.dst.WriteString(fmt.Sprintf("import \"%s\";\n", importFile))
	}
	if len(protoFile.Imports) > 0 {
		e.dst.WriteString("\n")
	}

	// Generate file-level options
	if len(protoFile.Options) > 0 {
		e.dst.WriteString("option ")
		for _, value := range protoFile.Options {
			e.encodeFieldOption(value)
		}
		e.dst.WriteString(";\n\n")
	}

	// Sort enums by name
	sort.Slice(protoFile.Enums, func(i, j int) bool {
		return protoFile.Enums[i].Name < protoFile.Enums[j].Name
	})

	// Generate enums
	for _, enum := range protoFile.Enums {
		e.encodeEnum(enum, 0)
	}

	// Sort messages by name
	sort.Slice(protoFile.Messages, func(i, j int) bool {
		return protoFile.Messages[i].Name < protoFile.Messages[j].Name
	})

	if len(protoFile.Messages) > 0 {
		for _, message := range protoFile.Messages {
			e.encodeMessage(message, 0)
		}
	}
	// Sort services by name
	sort.Slice(protoFile.Services, func(i, j int) bool {
		return protoFile.Services[i].Name < protoFile.Services[j].Name
	})

	// Generate services
	for _, service := range protoFile.Services {
		e.dst.WriteString(fmt.Sprintf("service %s {\n", service.Name))

		// Sort methods by name
		sort.Slice(service.Methods, func(i, j int) bool {
			return service.Methods[i].Name < service.Methods[j].Name
		})

		for _, method := range service.Methods {
			e.dst.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s)", method.Name, method.Input, method.Output))
			if len(method.Options) > 0 {
				e.dst.WriteString(" {\n")
				for _, option := range method.Options {
					e.dst.WriteString("     option ")
					e.encodeFieldOption(option)
					e.dst.WriteString(";\n")
				}
				e.dst.WriteString("  }\n")
			} else {
				e.dst.WriteString(";\n")
			}
		}
		e.dst.WriteString("}\n\n")
	}

	return e.dst.String()
}

// encodeEnum encodes enum types
func (e *Encoder) encodeEnum(enum *protobuf.ProtoEnum, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)
	e.dst.WriteString(fmt.Sprintf("%senum %s {\n", indent, enum.Name))

	// Generate enum values
	for _, value := range enum.Values {
		e.dst.WriteString(fmt.Sprintf("%s  %s = %d;\n", indent, value.Value, value.Index))
	}

	e.dst.WriteString(fmt.Sprintf("%s}\n\n", indent))
}

// encodeMessage recursively encodes messages, including nested messages and enums
func (e *Encoder) encodeMessage(message *protobuf.ProtoMessage, indentLevel int) {
	if indentLevel > 0 {
		e.dst.WriteString("\n")
	}
	indent := strings.Repeat("  ", indentLevel)
	e.dst.WriteString(fmt.Sprintf("%smessage %s {\n", indent, message.Name))

	// Generate message-level options
	if len(message.Options) > 0 {
		e.dst.WriteString(fmt.Sprintf("%s  option", indent))
		for _, option := range message.Options {
			e.encodeFieldOption(option)
			e.dst.WriteString(";\n")
		}
	}

	// Sort fields by name
	sort.Slice(message.Fields, func(i, j int) bool {
		return message.Fields[i].Name < message.Fields[j].Name
	})

	// Generate fields
	for i, field := range message.Fields {
		repeated := ""
		if field.Repeated {
			repeated = "repeated "
		}
		e.dst.WriteString(fmt.Sprintf("%s  %s%s %s = %d", indent, repeated, field.Type, field.Name, i+1))

		// Generate field-level options
		if len(field.Options) > 0 {
			e.dst.WriteString(" [\n    ")
			for j, option := range field.Options {
				e.encodeFieldOption(option)
				if j < len(field.Options)-1 {
					e.dst.WriteString(",\n    ")
				}
			}
			e.dst.WriteString("\n  ]")
		}
		e.dst.WriteString(";\n")
	}

	if len(message.Enums) > 0 {
		e.dst.WriteString("\n")
		// Recursively handle nested enums
		for _, nestedEnum := range message.Enums {
			e.encodeEnum(nestedEnum, indentLevel+1) // Increase indentation for nested enums
		}
	}
	// Recursively handle nested messages
	for _, nestedMessage := range message.Messages {
		e.encodeMessage(nestedMessage, indentLevel+1) // Increase indentation for nested messages
	}

	e.dst.WriteString(fmt.Sprintf("%s}\n\n", indent))
}

// encodeFieldOption encodes an option for a single field
func (e *Encoder) encodeFieldOption(opt *protobuf.Option) error {
	// Output the option name
	fmt.Fprintf(e.dst, "(%s) = ", opt.Name) // Add indentation for consistency

	// Check if the option value is a complex structure
	switch value := opt.Value.(type) {
	case map[string]interface{}:
		// If it's a map type, it needs to output as a nested structure
		fmt.Fprintf(e.dst, "{\n")        // Newline after {
		e.encodeFieldOptionMap(value, 6) // Output map content, passing the current indentation level
		fmt.Fprintf(e.dst, "    }")      // Indent and output the closing }, with the appropriate indentation level
	default:
		fmt.Fprintf(e.dst, "%s", value) // For simple types, output directly
	}

	return nil
}

// encodeFieldOptionMap encodes a complex map type option value
func (e *Encoder) encodeFieldOptionMap(optionMap map[string]interface{}, indent int) error {
	keys := make([]string, 0, len(optionMap))
	for k := range optionMap {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Sort keys to ensure consistent output order

	indentSpace := strings.Repeat(" ", indent) // Dynamically generate indent spaces

	for _, key := range keys {
		value := optionMap[key]
		// Output key-value pairs with appropriate indentation
		fmt.Fprintf(e.dst, "%s%s: %s", indentSpace, key, utils.Stringify(value)) // Add deeper indentation
		// Don't add a semicolon after the last item, maintain correct format
		fmt.Fprintf(e.dst, ";\n")
	}

	return nil
}
