package converter

import (
	"errors"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hertz-contrib/swagger-generate/swagger2idl/protobuf"
	"github.com/hertz-contrib/swagger-generate/swagger2idl/utils"
)

// ProtoConverter struct, used to convert OpenAPI specifications into Proto files
type ProtoConverter struct {
	spec            *openapi3.T
	ProtoFile       *protobuf.ProtoFile
	converterOption *ConvertOption
}

// ConvertOption adds a struct for conversion options
type ConvertOption struct {
	openapiOption bool
	apiOption     bool
	namingOption  bool
}

const (
	apiProtoFile     = "api.proto"
	openapiProtoFile = "openapi/annotations.proto"
	EmptyProtoFile   = "google/protobuf/empty.proto"

	EmptyMessage           = "google.protobuf.Empty"
	openapiDocumentOption  = "openapi.document"
	openapiOperationOption = "openapi.operation"
	openapiPropertyOption  = "openapi.property"
	openapiSchemaOption    = "openapi.schema"
)

var (
	methodToOption = map[string]string{
		"GET":     "api.get",
		"POST":    "api.post",
		"PUT":     "api.put",
		"PATCH":   "api.patch",
		"DELETE":  "api.delete",
		"HEAD":    "api.head",
		"OPTIONS": "api.options",
	}
)

// NewProtoConverter creates and initializes a ProtoConverter
func NewProtoConverter(spec *openapi3.T) *ProtoConverter {
	return &ProtoConverter{
		spec: spec,
		ProtoFile: &protobuf.ProtoFile{
			PackageName: utils.GetPackageName(spec),
			Messages:    []*protobuf.ProtoMessage{},
			Services:    []*protobuf.ProtoService{},
			Enums:       []*protobuf.ProtoEnum{},
			Imports:     []string{},
			Options:     []*protobuf.Option{},
		},
		converterOption: &ConvertOption{
			openapiOption: false,
			apiOption:     true,
			namingOption:  true,
		},
	}
}

// Convert converts the OpenAPI specification to a Proto file
func (c *ProtoConverter) Convert() error {

	// Convert the go Option to Proto

	// Convert components into Proto messages
	err := c.convertComponentsToProtoMessages(c.spec.Components)
	if err != nil {
		return fmt.Errorf("error converting components to proto messages: %w", err)
	}

	// Convert paths into Proto services
	err = c.convertPathsToProtoServices(c.spec.Paths)
	if err != nil {
		return fmt.Errorf("error converting paths to proto services: %w", err)
	}

	if c.converterOption.openapiOption {
		err = c.addOptionsToProto(c.spec)
		if err != nil {
			return fmt.Errorf("error parse options to proto: %w", err)
		}
	}

	return nil
}

// addOptionsToProto adds options to the ProtoFile
func (c *ProtoConverter) addOptionsToProto(spec *openapi3.T) error {
	optionStr := utils.StructToProtobuf(spec, "     ")

	schemaOption := &protobuf.Option{
		Name:  openapiDocumentOption,
		Value: optionStr,
	}
	c.ProtoFile.Options = append(c.ProtoFile.Options, schemaOption)
	c.AddProtoImport(openapiProtoFile)
	return nil
}

// convertComponentsToProtoMessages converts OpenAPI components into Proto messages and stores them in the ProtoFile
func (c *ProtoConverter) convertComponentsToProtoMessages(components *openapi3.Components) error {
	if components == nil {
		return nil
	}
	if components.Schemas == nil {
		return nil
	}
	for name, schemaRef := range components.Schemas {
		schema := schemaRef
		fieldOrMessage, err := c.ConvertSchemaToProtoType(schema, name, nil)
		if err != nil {
			return fmt.Errorf("error converting schema %s: %w", name, err)
		}
		switch v := fieldOrMessage.(type) {
		case *protobuf.ProtoField:
			message := &protobuf.ProtoMessage{
				Name:   name,
				Fields: []*protobuf.ProtoField{v},
			}

			if c.converterOption.openapiOption {
				optionStr := utils.StructToProtobuf(schema.Value, "    ")

				schemaOption := &protobuf.Option{
					Name:  openapiSchemaOption,
					Value: optionStr,
				}
				message.Options = append(message.Options, schemaOption)
				c.AddProtoImport(openapiProtoFile)

			}
			c.addMessageToProto(message)
		case *protobuf.ProtoMessage:
			if c.converterOption.openapiOption {
				optionStr := utils.StructToProtobuf(schema.Value, "    ")

				schemaOption := &protobuf.Option{
					Name:  openapiSchemaOption,
					Value: optionStr,
				}
				v.Options = append(v.Options, schemaOption)
				c.AddProtoImport(openapiProtoFile)

			}
			c.addMessageToProto(v)
		case *protobuf.ProtoEnum:
			if c.converterOption.openapiOption {
				optionStr := utils.StructToProtobuf(schema.Value, "    ")

				schemaOption := &protobuf.Option{
					Name:  openapiSchemaOption,
					Value: optionStr,
				}
				v.Options = append(v.Options, schemaOption)
				c.AddProtoImport(openapiProtoFile)

			}
			c.addEnumToProto(v)
		}
	}
	return nil
}

// convertPathsToProtoServices converts OpenAPI path items into Proto services and stores them in the ProtoFile
func (c *ProtoConverter) convertPathsToProtoServices(paths *openapi3.Paths) error {
	services, err := c.ConvertPathsToProtoServices(paths)
	if err != nil {
		return fmt.Errorf("error converting paths to proto services: %w", err)
	}

	c.ProtoFile.Services = append(c.ProtoFile.Services, services...)
	return nil
}

// ConvertPathsToProtoServices converts OpenAPI path items into Proto services
func (c *ProtoConverter) ConvertPathsToProtoServices(paths *openapi3.Paths) ([]*protobuf.ProtoService, error) {
	var services []*protobuf.ProtoService

	for path, pathItem := range paths.Map() {
		for method, operation := range pathItem.Operations() {
			serviceName := utils.GetServiceName(operation.Tags)

			methodName := utils.GenerateMethodName(operation, method)

			if c.converterOption.namingOption {
				methodName = utils.ToPascaleCase(methodName)
			} else {
				methodName = utils.ToUpperFirstLetter(methodName)
			}

			inputMessage, err := c.generateRequestMessage(operation)
			if err != nil {
				return nil, fmt.Errorf("error generating request message for %s: %w", methodName, err)
			}

			outputMessage, err := c.generateResponseMessage(operation)
			if err != nil {
				return nil, fmt.Errorf("error generating response message for %s: %w", methodName, err)
			}

			service := findOrCreateService(&services, serviceName)

			if !methodExistsInService(service, methodName) {
				protoMethod := &protobuf.ProtoMethod{
					Name:   methodName,
					Input:  inputMessage,
					Output: outputMessage,
				}

				if c.converterOption.apiOption {
					if optionName, ok := methodToOption[method]; ok {
						option := &protobuf.Option{
							Name:  optionName,
							Value: fmt.Sprintf("%q", utils.ConvertPath(path)),
						}
						protoMethod.Options = append(protoMethod.Options, option)
						c.AddProtoImport(apiProtoFile)
					}
				}

				if c.converterOption.openapiOption {
					optionStr := utils.StructToProtobuf(operation, "     ")

					schemaOption := &protobuf.Option{
						Name:  openapiOperationOption,
						Value: optionStr,
					}
					protoMethod.Options = append(protoMethod.Options, schemaOption)
					c.AddProtoImport(openapiProtoFile)

				}
				service.Methods = append(service.Methods, protoMethod)
			}
		}
	}

	return services, nil
}

// generateRequestMessage generates a request message for an operation
func (c *ProtoConverter) generateRequestMessage(operation *openapi3.Operation) (string, error) {
	messageName := operation.OperationID + "Request"
	if c.converterOption.namingOption {
		messageName = utils.ToPascaleCase(messageName)
	} else {
		messageName = utils.ToUpperFirstLetter(messageName)
	}
	message := &protobuf.ProtoMessage{Name: messageName}

	if operation.RequestBody == nil && len(operation.Parameters) == 0 {
		c.AddProtoImport(EmptyProtoFile)
		return EmptyMessage, nil
	}

	if operation.RequestBody != nil {
		if operation.RequestBody.Ref != "" {
			return utils.ExtractMessageNameFromRef(operation.RequestBody.Ref), nil
		}

		if operation.RequestBody.Value != nil && len(operation.RequestBody.Value.Content) > 0 {
			for mediaTypeStr, mediaType := range operation.RequestBody.Value.Content {
				schema := mediaType.Schema
				if schema != nil {
					protoType, err := c.ConvertSchemaToProtoType(schema, utils.FormatNaming(mediaTypeStr), message)
					if err != nil {
						return "", err
					}

					switch v := protoType.(type) {
					case *protobuf.ProtoField:
						if c.converterOption.apiOption {
							var optionName string
							if mediaTypeStr == "application/json" {
								optionName = "api.body"
							} else if mediaTypeStr == "application/x-www-form-urlencoded" || mediaTypeStr == "multipart/form-data" {
								optionName = "api.form"
							}
							if optionName != "" {
								v.Options = append(v.Options, &protobuf.Option{
									Name:  optionName,
									Value: fmt.Sprintf("%q", v.Name),
								})
								c.AddProtoImport(apiProtoFile)
							}
						}
						if c.converterOption.openapiOption {
							optionStr := utils.StructToProtobuf(schema.Value, "     ")

							schemaOption := &protobuf.Option{
								Name:  openapiPropertyOption,
								Value: optionStr,
							}
							v.Options = append(v.Options, schemaOption)
							c.AddProtoImport(openapiProtoFile)
						}
						addFieldIfNotExists(&message.Fields, v)
					case *protobuf.ProtoMessage:
						name := mediaTypeStr
						if c.converterOption.namingOption {
							name = utils.ToPascaleCase(name)
						} else {
							name = utils.FormatNaming(name)
						}
						newField := &protobuf.ProtoField{
							Name: name,
							Type: v.Name,
						}
						if c.converterOption.apiOption {
							var optionName string
							if mediaTypeStr == "application/json" {
								optionName = "api.body"
							} else if mediaTypeStr == "application/x-www-form-urlencoded" || mediaTypeStr == "multipart/form-data" {
								optionName = "api.form"
							}
							if optionName != "" {
								newField.Options = append(newField.Options, &protobuf.Option{
									Name:  optionName,
									Value: fmt.Sprintf("%q", v.Name),
								})
								c.AddProtoImport(apiProtoFile)
							}
						}
						if c.converterOption.openapiOption {
							optionStr := utils.StructToProtobuf(schema.Value, "     ")

							schemaOption := &protobuf.Option{
								Name:  openapiPropertyOption,
								Value: optionStr,
							}
							newField.Options = append(newField.Options, schemaOption)
							c.AddProtoImport(openapiProtoFile)
						}
						addMessageIfNotExists(&message.Messages, v)
						message.Fields = append(message.Fields, newField)
					case *protobuf.ProtoEnum:
						name := mediaTypeStr
						if c.converterOption.namingOption {
							name = utils.ToPascaleCase(name)
						} else {
							name = utils.FormatNaming(name)
						}
						newField := &protobuf.ProtoField{
							Name: name,
							Type: v.Name,
						}
						if c.converterOption.apiOption {
							var optionName string
							if mediaTypeStr == "application/json" {
								optionName = "api.body"
							} else if mediaTypeStr == "application/x-www-form-urlencoded" || mediaTypeStr == "multipart/form-data" {
								optionName = "api.form"
							}
							if optionName != "" {
								newField.Options = append(newField.Options, &protobuf.Option{
									Name:  optionName,
									Value: fmt.Sprintf("%q", v.Name),
								})
								c.AddProtoImport(apiProtoFile)
							}
						}
						if c.converterOption.openapiOption {
							optionStr := utils.StructToProtobuf(schema.Value, "     ")

							schemaOption := &protobuf.Option{
								Name:  openapiPropertyOption,
								Value: optionStr,
							}
							newField.Options = append(newField.Options, schemaOption)
							c.AddProtoImport(openapiProtoFile)
						}
						message.Enums = append(message.Enums, v)
						message.Fields = append(message.Fields, newField)
					}
				}
			}
		}
	}

	if len(operation.Parameters) > 0 {
		for _, param := range operation.Parameters {
			if param.Value.Schema != nil {
				fieldOrMessage, err := c.ConvertSchemaToProtoType(param.Value.Schema, param.Value.Name, message)
				if err != nil {
					return "", err
				}

				switch v := fieldOrMessage.(type) {
				case *protobuf.ProtoField:
					if c.converterOption.apiOption {
						v.Options = append(v.Options, &protobuf.Option{
							Name:  "api." + param.Value.In,
							Value: fmt.Sprintf("%q", param.Value.Name),
						})
						c.AddProtoImport(apiProtoFile)
					}
					if c.converterOption.openapiOption {
						optionStr := utils.StructToProtobuf(param.Value.Schema.Value, "     ")

						schemaOption := &protobuf.Option{
							Name:  openapiPropertyOption,
							Value: optionStr,
						}
						v.Options = append(v.Options, schemaOption)
						c.AddProtoImport(openapiProtoFile)
					}
					addFieldIfNotExists(&message.Fields, v)
				case *protobuf.ProtoMessage:
					name := param.Value.Name
					if c.converterOption.namingOption {
						name = utils.ToPascaleCase(name)
					} else {
						name = utils.ToUpperFirstLetter(name)
					}
					newField := &protobuf.ProtoField{
						Name: name,
						Type: v.Name,
					}
					if c.converterOption.apiOption {
						newField.Options = append(newField.Options, &protobuf.Option{
							Name:  "api." + param.Value.In,
							Value: fmt.Sprintf("%q", param.Value.Name),
						})
						c.AddProtoImport(apiProtoFile)
					}
					if c.converterOption.openapiOption {
						optionStr := utils.StructToProtobuf(param.Value.Schema.Value, "     ")

						schemaOption := &protobuf.Option{
							Name:  openapiPropertyOption,
							Value: optionStr,
						}
						newField.Options = append(newField.Options, schemaOption)
						c.AddProtoImport(openapiProtoFile)
					}
					addMessageIfNotExists(&message.Messages, v)
					message.Fields = append(message.Fields, newField)
				case *protobuf.ProtoEnum:
					name := param.Value.Name
					if c.converterOption.namingOption {
						name = utils.ToPascaleCase(name)
					} else {
						name = utils.ToUpperFirstLetter(name)
					}
					newField := &protobuf.ProtoField{
						Name: name,
						Type: v.Name,
					}
					if c.converterOption.apiOption {
						newField.Options = append(newField.Options, &protobuf.Option{
							Name:  "api." + param.Value.In,
							Value: fmt.Sprintf("%q", param.Value.Name),
						})
						c.AddProtoImport(apiProtoFile)
					}
					if c.converterOption.openapiOption {
						optionStr := utils.StructToProtobuf(param.Value.Schema.Value, "     ")

						schemaOption := &protobuf.Option{
							Name:  openapiPropertyOption,
							Value: optionStr,
						}
						newField.Options = append(newField.Options, schemaOption)
						c.AddProtoImport(openapiProtoFile)
					}
					message.Enums = append(message.Enums, v)
					message.Fields = append(message.Fields, newField)
				}
			}
		}
	}

	// if there are no fields or messages, return an empty message
	if len(message.Fields) > 0 || len(message.Messages) > 0 || len(message.Enums) > 0 {
		c.addMessageToProto(message)
		return message.Name, nil
	}

	return "", nil
}

// generateResponseMessage generates a response message for an operation
func (c *ProtoConverter) generateResponseMessage(operation *openapi3.Operation) (string, error) {
	if operation.Responses == nil {
		return "", nil
	}

	responses := operation.Responses.Map()
	responseCount := 0
	for _, responseRef := range responses {
		if responseRef.Ref == "" && (responseRef.Value == nil || (len(responseRef.Value.Content) == 0 && len(responseRef.Value.Headers) == 0)) {
			continue
		}
		responseCount++
	}

	if responseCount == 1 {
		for statusCode, responseRef := range responses {
			if responseRef.Ref == "" && (responseRef.Value == nil || (len(responseRef.Value.Content) == 0 && len(responseRef.Value.Headers) == 0)) {
				c.AddProtoImport(EmptyProtoFile)
				return EmptyMessage, nil
			}
			return c.processSingleResponse(statusCode, responseRef, operation)
		}
	}

	// create a wrapper message for multiple responses
	wrapperMessageName := operation.OperationID + "Response"
	if c.converterOption.namingOption {
		wrapperMessageName = utils.ToPascaleCase(wrapperMessageName)
	} else {
		wrapperMessageName = utils.ToUpperFirstLetter(wrapperMessageName)
	}
	wrapperMessage := &protobuf.ProtoMessage{Name: wrapperMessageName}

	emptyFlag := true

	for statusCode, responseRef := range responses {
		if responseRef.Ref == "" && (responseRef.Value == nil || len(responseRef.Value.Content) == 0) {
			break
		}
		emptyFlag = false
		messageName, err := c.processSingleResponse(statusCode, responseRef, operation)
		if err != nil {
			return "", err
		}

		name := "Response_" + statusCode
		if c.converterOption.namingOption {
			name = utils.ToPascaleCase(name)
		} else {
			name = utils.ToUpperFirstLetter(name)
		}
		field := &protobuf.ProtoField{
			Name: name,
			Type: messageName,
		}
		wrapperMessage.Fields = append(wrapperMessage.Fields, field)
	}

	if emptyFlag {
		c.AddProtoImport(EmptyProtoFile)
		return EmptyMessage, nil
	}

	c.addMessageToProto(wrapperMessage)

	return wrapperMessageName, nil
}

// processSingleResponse deals with a single response in an operation
func (c *ProtoConverter) processSingleResponse(statusCode string, responseRef *openapi3.ResponseRef, operation *openapi3.Operation) (string, error) {
	if responseRef.Ref != "" {
		return utils.ExtractMessageNameFromRef(responseRef.Ref), nil
	}

	response := responseRef.Value
	messageName := operation.OperationID + "Response" + utils.ToUpperFirstLetter(statusCode)
	if c.converterOption.namingOption {
		messageName = utils.ToPascaleCase(messageName)
	} else {
		messageName = utils.ToUpperFirstLetter(messageName)
	}
	newMessage := &protobuf.ProtoMessage{Name: messageName}

	if len(response.Headers) > 0 {
		for headerName, headerRef := range response.Headers {
			if headerRef != nil {

				fieldOrMessage, err := c.ConvertSchemaToProtoType(headerRef.Value.Schema, headerName, newMessage)
				if err != nil {
					return "", err
				}

				switch v := fieldOrMessage.(type) {
				case *protobuf.ProtoField:
					if c.converterOption.apiOption {
						option := &protobuf.Option{
							Name:  "api.header",
							Value: fmt.Sprintf("%q", headerName),
						}
						v.Options = append(v.Options, option)
						c.AddProtoImport(apiProtoFile)
					}
					addFieldIfNotExists(&newMessage.Fields, v)
				case *protobuf.ProtoMessage:
					name := headerName
					if c.converterOption.namingOption {
						name = utils.ToSnakeCase(name)
					} else {
						name = utils.ToUpperFirstLetter(name)
					}
					newField := &protobuf.ProtoField{
						Name: name,
						Type: v.Name,
					}
					if c.converterOption.apiOption {
						option := &protobuf.Option{
							Name:  "api.header",
							Value: fmt.Sprintf("%q", headerName),
						}
						newField.Options = append(newField.Options, option)
						c.AddProtoImport(apiProtoFile)
					}
					addMessageIfNotExists(&newMessage.Messages, v)
					newMessage.Fields = append(newMessage.Fields, newField)
				case *protobuf.ProtoEnum:
					name := headerName
					if c.converterOption.namingOption {
						name = utils.ToSnakeCase(name)
					} else {
						name = utils.ToUpperFirstLetter(name)
					}
					newField := &protobuf.ProtoField{
						Name: name,
						Type: v.Name,
					}
					if c.converterOption.apiOption {
						option := &protobuf.Option{
							Name:  "api.header",
							Value: fmt.Sprintf("%q", headerName),
						}
						newField.Options = append(newField.Options, option)
						c.AddProtoImport(apiProtoFile)
					}
					newMessage.Enums = append(newMessage.Enums, v)
					newMessage.Fields = append(newMessage.Fields, newField)
				}
			}
		}
	}

	for mediaTypeStr, mediaType := range response.Content {
		schema := mediaType.Schema
		if schema != nil {

			fieldOrMessage, err := c.ConvertSchemaToProtoType(schema, utils.FormatNaming(mediaTypeStr), newMessage)
			if err != nil {
				return "", err
			}

			switch v := fieldOrMessage.(type) {
			case *protobuf.ProtoField:
				if c.converterOption.apiOption && mediaTypeStr == "application/json" {
					option := &protobuf.Option{
						Name:  "api.body",
						Value: fmt.Sprintf("%q", v.Name),
					}
					v.Options = append(v.Options, option)
					c.AddProtoImport(apiProtoFile)
				}
				if c.converterOption.openapiOption {
					optionStr := utils.StructToProtobuf(schema.Value, "     ")

					schemaOption := &protobuf.Option{
						Name:  openapiPropertyOption,
						Value: optionStr,
					}
					v.Options = append(v.Options, schemaOption)
					c.AddProtoImport(openapiProtoFile)
				}
				addFieldIfNotExists(&newMessage.Fields, v)
			case *protobuf.ProtoMessage:
				addMessageIfNotExists(&newMessage.Messages, v)
				name := mediaTypeStr
				if c.converterOption.namingOption {
					name = utils.ToSnakeCase(mediaTypeStr)
				} else {
					name = utils.ToUpperFirstLetter(name)
				}
				newField := &protobuf.ProtoField{
					Name: name,
					Type: v.Name,
				}
				if c.converterOption.apiOption && mediaTypeStr == "application/json" {
					option := &protobuf.Option{
						Name:  "api.body",
						Value: fmt.Sprintf("%q", v.Name),
					}
					newField.Options = append(newField.Options, option)
					c.AddProtoImport(apiProtoFile)
				}
				if c.converterOption.openapiOption {
					optionStr := utils.StructToProtobuf(schema.Value, "     ")

					schemaOption := &protobuf.Option{
						Name:  openapiPropertyOption,
						Value: optionStr,
					}
					newField.Options = append(newField.Options, schemaOption)
					c.AddProtoImport(openapiProtoFile)
				}
				newMessage.Fields = append(newMessage.Fields, newField)
			case *protobuf.ProtoEnum:
				name := mediaTypeStr
				if c.converterOption.namingOption {
					name = utils.ToSnakeCase(mediaTypeStr)
				} else {
					name = utils.ToUpperFirstLetter(name)
				}
				newField := &protobuf.ProtoField{
					Name: name,
					Type: v.Name,
				}
				if c.converterOption.apiOption && mediaTypeStr == "application/json" {
					option := &protobuf.Option{
						Name:  "api.body",
						Value: fmt.Sprintf("%q", v.Name),
					}
					newField.Options = append(newField.Options, option)
					c.AddProtoImport(apiProtoFile)
				}
				if c.converterOption.openapiOption {
					optionStr := utils.StructToProtobuf(schema.Value, "     ")

					schemaOption := &protobuf.Option{
						Name:  openapiPropertyOption,
						Value: optionStr,
					}
					newField.Options = append(newField.Options, schemaOption)
					c.AddProtoImport(openapiProtoFile)
				}
				newMessage.Enums = append(newMessage.Enums, v)
				newMessage.Fields = append(newMessage.Fields, newField)
			}
		}
	}

	if len(newMessage.Fields) > 0 || len(newMessage.Messages) > 0 || len(newMessage.Enums) > 0 {
		c.addMessageToProto(newMessage)
		return newMessage.Name, nil
	}
	return "", nil
}

// ConvertSchemaToProtoType converts an OpenAPI schema to a Proto field or message
func (c *ProtoConverter) ConvertSchemaToProtoType(
	schemaRef *openapi3.SchemaRef,
	protoName string,
	parentMessage *protobuf.ProtoMessage) (interface{}, error) {

	var protoType string
	var result interface{}

	// Handle referenced schema
	if schemaRef.Ref != "" {
		name := c.applyNamingOption(utils.ExtractMessageNameFromRef(schemaRef.Ref))
		return &protobuf.ProtoField{
			Name: name,
			Type: utils.ExtractMessageNameFromRef(schemaRef.Ref),
		}, nil
	}

	// Ensure schema value is valid
	if schemaRef.Value == nil || schemaRef.Value.Type == nil {
		return nil, errors.New("schema type is required")
	}

	schema := schemaRef.Value

	// Process schema type
	switch {
	case schema.Type.Includes("string"):
		if schema.Format == "date" || schema.Format == "date-time" {
			protoType = "google.protobuf.Timestamp"
			c.AddProtoImport("google/protobuf/timestamp.proto")
		} else if len(schema.Enum) != 0 {
			var name string
			if parentMessage == nil {
				name = protoName
			} else {
				name = c.applyNamingOption(parentMessage.Name + utils.ToUpperFirstLetter(protoName))
			}
			protoEnum := &protobuf.ProtoEnum{
				Name: name,
			}
			for i, enumValue := range schema.Enum {
				protoEnum.Values = append(protoEnum.Values, &protobuf.ProtoEnumValue{
					Index: i,
					Value: enumValue,
				})
			}
			result = protoEnum
		} else {
			protoType = "string"
		}

	case schema.Type.Includes("integer"):
		if len(schema.Enum) != 0 {
			var name string
			if parentMessage == nil {
				name = protoName
			} else {
				name = c.applyNamingOption(parentMessage.Name + utils.ToUpperFirstLetter(protoName))
			}
			protoEnum := &protobuf.ProtoEnum{
				Name: name,
			}
			for i, enumValue := range schema.Enum {
				protoEnum.Values = append(protoEnum.Values, &protobuf.ProtoEnumValue{
					Index: i,
					Value: enumValue,
				})
			}
			result = protoEnum
		} else if schema.Format == "int32" {
			protoType = "int32"
		} else {
			protoType = "int64"
		}

	case schema.Type.Includes("number"):
		if len(schema.Enum) != 0 {
			var name string
			if parentMessage == nil {
				name = protoName
			} else {
				name = c.applyNamingOption(parentMessage.Name + utils.ToUpperFirstLetter(protoName))
			}
			protoEnum := &protobuf.ProtoEnum{
				Name: name,
			}
			for i, enumValue := range schema.Enum {
				protoEnum.Values = append(protoEnum.Values, &protobuf.ProtoEnumValue{
					Index: i,
					Value: enumValue,
				})
			}
			result = protoEnum
		} else if schema.Format == "float" {
			protoType = "float"
		} else {
			protoType = "double"
		}

	case schema.Type.Includes("boolean"):
		protoType = "bool"

	case schema.Type.Includes("array"):
		if schema.Items != nil {
			fieldOrMessage, err := c.ConvertSchemaToProtoType(schema.Items, protoName+"Item", parentMessage)
			if err != nil {
				return nil, err
			}

			fieldType := ""
			if field, ok := fieldOrMessage.(*protobuf.ProtoField); ok {
				fieldType = field.Type
			} else if nestedMessage, ok := fieldOrMessage.(*protobuf.ProtoMessage); ok {
				fieldType = nestedMessage.Name
				c.addNestedMessageToParent(parentMessage, nestedMessage)
			} else if enum, ok := fieldOrMessage.(*protobuf.ProtoEnum); ok {
				fieldType = enum.Name
				c.addNestedEnumToParent(parentMessage, enum)
			}

			result = &protobuf.ProtoField{
				Name:     c.applyNamingOption(protoName),
				Type:     fieldType,
				Repeated: true,
			}
		}

	case schema.Type.Includes("object"):
		var message *protobuf.ProtoMessage
		if parentMessage == nil {
			message = &protobuf.ProtoMessage{Name: protoName}
		} else {
			message = &protobuf.ProtoMessage{Name: c.applyNamingOption(parentMessage.Name + utils.ToUpperFirstLetter(protoName))}
		}
		for propName, propSchema := range schema.Properties {
			protoType, err := c.ConvertSchemaToProtoType(propSchema, propName, message)
			if err != nil {
				return nil, err
			}

			if field, ok := protoType.(*protobuf.ProtoField); ok {
				message.Fields = append(message.Fields, field)
			} else if nestedMessage, ok := protoType.(*protobuf.ProtoMessage); ok {
				//nestedMessage.Name = message.Name + nestedMessage.Name
				c.addNestedMessageToParent(message, nestedMessage)
				message.Fields = append(message.Fields, &protobuf.ProtoField{
					Name: c.applyNamingOption(propName),
					Type: nestedMessage.Name,
				})
			} else if enum, ok := protoType.(*protobuf.ProtoEnum); ok {
				//enum.Name = message.Name + enum.Name
				c.addNestedEnumToParent(message, enum)
				message.Fields = append(message.Fields, &protobuf.ProtoField{
					Name: c.applyNamingOption(propName),
					Type: enum.Name,
				})
			}
		}

		if schema.AdditionalProperties.Schema != nil {
			mapValueType := "string"
			additionalPropMessage, err := c.ConvertSchemaToProtoType(schema.AdditionalProperties.Schema, protoName+"AdditionalProperties", parentMessage)
			if err != nil {
				return nil, err
			}
			if msg, ok := additionalPropMessage.(*protobuf.ProtoMessage); ok {
				mapValueType = msg.Name
			} else if enum, ok := additionalPropMessage.(*protobuf.ProtoEnum); ok {
				mapValueType = enum.Name
			}

			message.Fields = append(message.Fields, &protobuf.ProtoField{
				Name: "additionalProperties",
				Type: "map<string, " + mapValueType + ">",
			})
		}

		result = message
	}

	// If result is still nil, construct a default ProtoField
	if result == nil {
		result = &protobuf.ProtoField{
			Name: c.applyNamingOption(protoName),
			Type: protoType,
		}
	}

	return result, nil
}

// applyNamingOption applies naming convention based on the converter's naming option
func (c *ProtoConverter) applyNamingOption(name string) string {
	if c.converterOption.namingOption {
		return utils.ToPascaleCase(name)
	}
	return utils.ToUpperFirstLetter(name)
}

// addNestedMessageToParent adds a nested message to a parent message
func (c *ProtoConverter) addNestedMessageToParent(parentMessage, nestedMessage *protobuf.ProtoMessage) {
	if parentMessage != nil && nestedMessage != nil {
		parentMessage.Messages = append(parentMessage.Messages, nestedMessage)
	}
}

// addNestedEnum adds a nested Enum to a parent message
func (c *ProtoConverter) addNestedEnumToParent(parentMessage *protobuf.ProtoMessage, nestedEnum *protobuf.ProtoEnum) {
	if parentMessage != nil && nestedEnum != nil {
		parentMessage.Enums = append(parentMessage.Enums, nestedEnum)
	}
}

// mergeProtoMessage merges a ProtoMessage into the ProtoFile
func (c *ProtoConverter) addMessageToProto(message *protobuf.ProtoMessage) error {
	var existingMessage *protobuf.ProtoMessage
	for _, msg := range c.ProtoFile.Messages {
		if msg.Name == message.Name {
			existingMessage = msg
			break
		}
	}

	// merge message
	if existingMessage != nil {
		// merge Fields
		fieldNames := make(map[string]struct{})
		for _, field := range existingMessage.Fields {
			fieldNames[field.Name] = struct{}{} // 记录已有字段名称
		}
		for _, newField := range message.Fields {
			if _, exists := fieldNames[newField.Name]; !exists {
				existingMessage.Fields = append(existingMessage.Fields, newField)
			}
		}

		// merge Messages
		messageNames := make(map[string]struct{})
		for _, nestedMsg := range existingMessage.Messages {
			messageNames[nestedMsg.Name] = struct{}{}
		}
		for _, newMessage := range message.Messages {
			if _, exists := messageNames[newMessage.Name]; !exists {
				existingMessage.Messages = append(existingMessage.Messages, newMessage)
			}
		}

		// merge Enums
		enumNames := make(map[string]struct{})
		for _, enum := range existingMessage.Enums {
			enumNames[enum.Name] = struct{}{}
		}
		for _, newEnum := range message.Enums {
			if _, exists := enumNames[newEnum.Name]; !exists {
				existingMessage.Enums = append(existingMessage.Enums, newEnum)
			}
		}

		// merge Options
		optionNames := make(map[string]struct{})
		for _, option := range existingMessage.Options {
			optionNames[option.Name] = struct{}{}
		}
		for _, newOption := range message.Options {
			if _, exists := optionNames[newOption.Name]; !exists {
				existingMessage.Options = append(existingMessage.Options, newOption)
			}
		}
	} else {
		c.ProtoFile.Messages = append(c.ProtoFile.Messages, message)
	}

	return nil
}

// addEnumToProto adds an enum to the ProtoFile
func (c *ProtoConverter) addEnumToProto(enum *protobuf.ProtoEnum) {
	c.ProtoFile.Enums = append(c.ProtoFile.Enums, enum)
}

// AddProtoImport adds an import to the ProtoFile
func (c *ProtoConverter) AddProtoImport(importFile string) {
	if c.ProtoFile != nil {
		for _, existingImport := range c.ProtoFile.Imports {
			if existingImport == importFile {
				return
			}
		}
		c.ProtoFile.Imports = append(c.ProtoFile.Imports, importFile)
	}
}

// addFieldIfNotExists adds a field to Fields if it does not already exist
func addFieldIfNotExists(fields *[]*protobuf.ProtoField, field *protobuf.ProtoField) {
	for _, existingField := range *fields {
		if existingField.Name == field.Name {
			return
		}
	}
	*fields = append(*fields, field)
}

// addMessageIfNotExists adds a message to Messages if it does not already exist
func addMessageIfNotExists(messages *[]*protobuf.ProtoMessage, nestedMessage *protobuf.ProtoMessage) {
	for _, existingMessage := range *messages {
		if existingMessage.Name == nestedMessage.Name {
			return
		}
	}
	*messages = append(*messages, nestedMessage)
}

// methodExistsInService checks if a method exists in a service
func methodExistsInService(service *protobuf.ProtoService, methodName string) bool {
	for _, method := range service.Methods {
		if method.Name == methodName {
			return true
		}
	}
	return false
}

// findOrCreateService finds or creates a service
func findOrCreateService(services *[]*protobuf.ProtoService, serviceName string) *protobuf.ProtoService {
	for i := range *services {
		if (*services)[i].Name == serviceName {
			return (*services)[i]
		}
	}

	newService := &protobuf.ProtoService{Name: serviceName}
	*services = append(*services, newService)
	return (*services)[len(*services)-1]
}
