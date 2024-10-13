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

package utils

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
)

func Stringify(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v) // Add quotes around strings
	case int, int64, float64:
		return fmt.Sprintf("%v", v) // Output numbers directly
	case *uint64:
		return fmt.Sprintf("%d", *v) // Handle *uint64 pointer type
	case []string:
		return fmt.Sprintf("[%s]", strings.Join(v, ", ")) // Output string arrays as a list
	case []interface{}:
		// Handle arrays of arbitrary types
		var strValues []string
		for _, item := range v {
			strValues = append(strValues, Stringify(item))
		}
		return fmt.Sprintf("[%s]", strings.Join(strValues, ", "))
	default:
		return fmt.Sprintf("%v", v) // Convert other types directly to string
	}
}

func StructToOption(value interface{}, indent string) string {
	var sb strings.Builder
	v := reflect.ValueOf(value)
	t := reflect.TypeOf(value)

	// If it's a pointer, get the actual value
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "" // Skip nil pointers
		}
		v = v.Elem()
		t = t.Elem()
	}

	// Handle slice types
	if v.Kind() == reflect.Slice {
		if v.Len() == 0 {
			return "" // Skip empty slices
		}
		sb.WriteString("[\n")
		for i := 0; i < v.Len(); i++ {
			sb.WriteString(fmt.Sprintf("%s  ", indent))
			sb.WriteString(StructToOption(v.Index(i).Interface(), indent+"  "))
			if i < v.Len()-1 {
				sb.WriteString(",\n")
			}
		}
		sb.WriteString(fmt.Sprintf("\n%s]", indent))
		return sb.String()
	}

	// Handle map types
	if v.Kind() == reflect.Map {
		if v.Len() == 0 {
			return "" // Skip empty maps
		}
		sb.WriteString("{\n")
		for _, key := range v.MapKeys() {
			if isZeroValue(v.MapIndex(key)) {
				continue
			}
			sb.WriteString(fmt.Sprintf("%s  %v: ", indent, reflect.ValueOf(ToSnakeCase(key.String()))))
			sb.WriteString(StructToOption(v.MapIndex(key).Interface(), indent+"  "))
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf("%s}", indent))
		return sb.String()
	}

	// Handle struct types
	if v.Kind() == reflect.Struct {
		sb.WriteString("{\n")
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := t.Field(i)

			// Skip unexported fields
			if !field.CanInterface() {
				continue
			}

			// Skip fields with zero values
			if isZeroValue(field) {
				continue
			}

			fieldName := fieldType.Tag.Get("json")
			if fieldName == "" {
				fieldName = fieldType.Name // If no json tag, use field name
			}
			fieldName = strings.Split(fieldName, ",")[0] // Remove options from json tag, e.g., "omitempty"

			// Skip specific fields (Parameters, RequestBody, Responses)
			if fieldName == "parameters" || fieldName == "requestBody" || fieldName == "responses" ||
				fieldName == "schemas" || fieldName == "requestBodies" || fieldName == "items" ||
				fieldName == "paths" || fieldName == "properties" || fieldName == "content" ||
				fieldName == "schema" || fieldName == "oneOf" || fieldName == "allOf" || fieldName == "anyOf" ||
				fieldName == "additionalProperties" || fieldName == "-" ||
				fieldName == "components" {
				continue
			}

			fieldName = ToSnakeCase(fieldName) // Convert field name to snake_case

			// Use the field name as the Protobuf key
			sb.WriteString(fmt.Sprintf("%s  %s: ", indent, fieldName))

			// Recursively handle the field
			sb.WriteString(StructToOption(field.Interface(), indent+"  "))
			sb.WriteString(";\n")
		}
		sb.WriteString(fmt.Sprintf("%s}", indent))
		return sb.String()
	}

	// Handle other basic types
	switch v.Kind() {
	case reflect.String:
		if v.String() == "" {
			return "" // Skip empty strings
		}

		// Process multi-line strings by replacing actual newlines with "\n"
		multiLineStr := strings.ReplaceAll(v.String(), "\n", "\\n")
		return fmt.Sprintf("\"%s\"", multiLineStr)
	case reflect.Int, reflect.Int64, reflect.Int32:
		if v.Int() == 0 {
			return "" // Skip 0 values
		}
		return fmt.Sprintf("%d", v.Int())
	case reflect.Float64:
		if v.Float() == 0 {
			return "" // Skip 0.0
		}
		return fmt.Sprintf("%f", v.Float())
	case reflect.Bool:
		if !v.Bool() {
			return "" // Skip false
		}
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Ptr:
		if !v.IsNil() {
			return StructToOption(v.Interface(), indent)
		}
		return ""
	default:
		// Skip zero values
		if !v.IsValid() || v.IsZero() {
			return ""
		}
		return fmt.Sprintf("\"%v\"", v.Interface())
	}
}

func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Slice, reflect.Array:
		return v.Len() == 0 // Check if slice or array is empty
	case reflect.Map:
		if v.Len() == 0 {
			return true
		}
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			if !isZeroValue(value) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isZeroValue(v.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return !v.IsValid()
	}
}

func GetMethodName(operation *openapi3.Operation, path, method string) string {
	if operation.OperationID != "" {
		return operation.OperationID
	}
	if operation.Tags != nil {
		return operation.Tags[0]
	}
	if path != "" {
		// Convert path to PascalCase, replacing placeholders with a suitable format
		convertedPath := ConvertPathToPascalCase(path)
		return convertedPath + strings.Title(strings.ToLower(method))
	}
	// If no OperationID, generate using HTTP method
	return strings.Title(strings.ToLower(method)) + "Method"
}

func GetServiceName(operation *openapi3.Operation) string {
	if len(operation.Tags) > 0 {
		return operation.Tags[0]
	}
	return "DefaultService"
}

func GetMessageName(operation *openapi3.Operation, methodName, suffix string) string {
	if operation.OperationID != "" {
		return operation.OperationID + suffix
	}
	return methodName + suffix
}

func GetPackageName(spec *openapi3.T) string {
	if spec.Info.Title != "" {
		return ToSnakeCase(spec.Info.Title)
	}
	if spec.Info.Description != "" {
		return ToSnakeCase(spec.Info.Description)
	}
	return "default_package"
}

// ConvertPathToPascalCase converts a path with placeholders to PascalCase
func ConvertPathToPascalCase(path string) string {
	// Replace placeholders like {orderId} with OrderId
	re := regexp.MustCompile(`\{(\w+)\}`)
	path = re.ReplaceAllStringFunc(path, func(s string) string {
		return ToPascaleCase(strings.Trim(s, "{}"))
	})

	// Split the path by '/' and convert each segment to PascalCase
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = ToPascaleCase(segment)
	}

	// Join the segments back together
	return strings.Join(segments, "")
}

func ExtractMessageNameFromRef(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1] // Return the last part, usually the name of the reference
}

func ConvertPath(path string) string {
	// Regular expression to match content inside {}
	re := regexp.MustCompile(`\{(\w+)\}`)
	// Replace {param} with :param
	result := re.ReplaceAllString(path, ":$1")
	return result
}

// ToUpperCase converts the first letter of a string to uppercase
func ToUpperCase(s string) string {
	if len(s) == 0 {
		return s
	}

	firstChar := unicode.ToUpper(rune(s[0]))

	if len(s) == 1 {
		return string(firstChar)
	}

	return string(firstChar) + s[1:]
}

func FormatStr(str string) string {
	str = strings.ReplaceAll(str, " ", "_")
	str = strings.ReplaceAll(str, "/", "_")
	str = strings.ReplaceAll(str, "-", "_")
	reg, _ := regexp.Compile(`[^a-zA-Z0-9_]`)
	str = reg.ReplaceAllString(str, "")
	return str
}

func ToCamelCase(name string) string {
	name = strcase.ToCamel(name)
	return name
}

func ToPascaleCase(name string) string {
	name = strcase.ToCamel(name)
	name = ToUpperCase(name)
	return name
}

func ToUpperSnakeCase(name string) string {
	name = FormatStr(name)
	name = ToSnake(name)
	name = strings.ToUpper(name)
	return name
}

func ToSnakeCase(name string) string {
	name = FormatStr(name)
	name = ToSnake(name)
	return name
}

// ToSnake converts a string to snake_case
func ToSnake(s string) string {
	return ToDelimited(s, '_')
}

// ToDelimited converts a string to delimited.snake.case
// (in this case `delimiter = '.'`)
func ToDelimited(s string, delimiter uint8) string {
	return ToScreamingDelimited(s, delimiter, "", false)
}

// ToScreamingDelimited converts a string to SCREAMING.DELIMITED.SNAKE.CASE
// (in this case `delimiter = '.'; screaming = true`)
// or delimited.snake.case
// (in this case `delimiter = '.'; screaming = false`)
func ToScreamingDelimited(s string, delimiter uint8, ignore string, screaming bool) string {
	s = strings.TrimSpace(s)
	n := strings.Builder{}
	n.Grow(len(s) + 2) // nominal 2 bytes of extra space for inserted delimiters
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if vIsLow && screaming {
			v += 'A'
			v -= 'a'
		} else if vIsCap && !screaming {
			v += 'a'
			v -= 'A'
		}

		// treat acronyms as words, eg for JSONData -> JSON is a whole word
		if i+1 < len(s) {
			next := s[i+1]
			vIsNum := v >= '0' && v <= '9'
			nextIsCap := next >= 'A' && next <= 'Z'
			nextIsLow := next >= 'a' && next <= 'z'
			nextIsNum := next >= '0' && next <= '9'

			// add delimiter if the next character is of a different type
			// but do not insert delimiter between a letter and a number
			if (vIsCap && (nextIsLow || nextIsNum)) || (vIsLow && (nextIsCap || nextIsNum)) || (vIsNum && (nextIsCap || nextIsLow)) {
				prevIgnore := ignore != "" && i > 0 && strings.ContainsAny(string(s[i-1]), ignore)
				if !prevIgnore {
					if vIsCap && nextIsLow {
						if prevIsCap := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'; prevIsCap {
							n.WriteByte(delimiter)
						}
					}

					// Skip adding delimiter if current character is a letter followed by a number
					if !(vIsLow && nextIsNum) && !(vIsCap && nextIsNum) {
						n.WriteByte(v)
						if vIsLow || vIsNum || nextIsNum {
							n.WriteByte(delimiter)
						}
						continue
					}
				}
			}
		}

		if (v == ' ' || v == '_' || v == '-' || v == '.') && !strings.ContainsAny(string(v), ignore) {
			// replace space/underscore/hyphen/dot with delimiter
			n.WriteByte(delimiter)
		} else {
			n.WriteByte(v)
		}
	}

	return n.String()
}
