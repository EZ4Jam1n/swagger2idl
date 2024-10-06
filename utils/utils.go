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

func StructToProtobuf(value interface{}, indent string) string {
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
			sb.WriteString(StructToProtobuf(v.Index(i).Interface(), indent+"  "))
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
			sb.WriteString(fmt.Sprintf("%s  %v: ", indent, key))
			sb.WriteString(StructToProtobuf(v.MapIndex(key).Interface(), indent+"  "))
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
				fieldName == "paths" || fieldName == "properties" {
				continue
			}

			// Use the field name as the Protobuf key
			sb.WriteString(fmt.Sprintf("%s  %s: ", indent, fieldName))

			// Recursively handle the field
			sb.WriteString(StructToProtobuf(field.Interface(), indent+"  "))
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
			return StructToProtobuf(v.Interface(), indent)
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
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Struct:
		// Recursively check if all fields in the struct are zero values
		for i := 0; i < v.NumField(); i++ {
			if !isZeroValue(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		return !v.IsValid()
	}
}

func GenerateMethodName(operation *openapi3.Operation, method string) string {
	if operation.OperationID != "" {
		return operation.OperationID
	}
	if operation.Tags != nil {
		return operation.Tags[0]
	}
	if operation.Summary != "" {
		return operation.Summary
	}
	if operation.Description != "" {
		return operation.Description
	}
	// If no OperationID, generate using HTTP method
	return strings.Title(strings.ToLower(method)) + "Method"
}

func ExtractMessageNameFromRef(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1] // Return the last part, usually the name of the reference
}

func GetServiceName(tags []string) string {
	if len(tags) > 0 {
		return tags[0]
	}
	return "DefaultService"
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

func ConvertPath(path string) string {
	// Regular expression to match content inside {}
	re := regexp.MustCompile(`\{(\w+)\}`)
	// Replace {param} with :param
	result := re.ReplaceAllString(path, ":$1")
	return result
}

// ToUpperFirstLetter converts the first letter of a string to uppercase
func ToUpperFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}

	firstChar := unicode.ToUpper(rune(s[0]))

	if len(s) == 1 {
		return string(firstChar)
	}

	return string(firstChar) + s[1:]
}

func FormatNaming(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return name
}

func ToPascaleCase(name string) string {
	name = strcase.ToCamel(name)
	name = ToUpperFirstLetter(name)
	return name
}

func ToSnakeCase(name string) string {
	name = FormatNaming(name)
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
