package converter

import (
	"errors"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hertz-contrib/swagger-generate/swagger2idl/thrift"
	"github.com/hertz-contrib/swagger-generate/swagger2idl/utils"
)

const (
	openapiThriftFile = "openapi.thrift"
)

// ThriftConverter struct, used to convert OpenAPI specifications into Thrift files
type ThriftConverter struct {
	spec            *openapi3.T
	ThriftFile      *thrift.ThriftFile
	converterOption *ConvertOption
}

// NewThriftConverter creates and initializes a ThriftConverter
func NewThriftConverter(spec *openapi3.T, option *ConvertOption) *ThriftConverter {
	return &ThriftConverter{
		spec: spec,
		ThriftFile: &thrift.ThriftFile{
			Namespace: map[string]string{},
			Includes:  []string{},
			Structs:   []*thrift.ThriftStruct{},
			Enums:     []*thrift.ThriftEnum{},
			Constants: []*thrift.ThriftConstant{},
			Services:  []*thrift.ThriftService{},
		},
		converterOption: option,
	}
}

// Convert converts the OpenAPI specification to a Thrift file
func (c *ThriftConverter) Convert() error {
	// Convert the go Option to Thrift
	err := c.addExtensionsToProtoOptions()
	if err != nil {
		return fmt.Errorf("error parsing extensions to proto options: %w", err)
	}

	// Convert tags into Thrift services
	err = c.convertTagsToThriftServices()
	if err != nil {
		return fmt.Errorf("error converting tags to thrift services: %w", err)
	}

	// Convert components into Thrift messages
	err = c.convertComponentsToThriftMessages()
	if err != nil {
		return fmt.Errorf("error converting components to thrift messages: %w", err)
	}

	// Convert paths into Thrift services
	err = c.convertPathsToThriftServices()
	if err != nil {
		return fmt.Errorf("error converting paths to thrift services: %w", err)
	}

	if c.converterOption.OpenapiOption {
		err = c.addOptionsToThrift()
		if err != nil {
			return fmt.Errorf("error adding options to thrift: %w", err)
		}
	}

	return nil
}

func (c *ThriftConverter) GetIdl() interface{} {
	return c.ThriftFile
}

// addOptionsToThrift adds options to the Thrift file
func (c *ThriftConverter) addOptionsToThrift() error {
	if len(c.ThriftFile.Services) > 0 {
		if c.converterOption.OpenapiOption {
			optionStr := utils.StructToOption(c.spec, "")

			schemaOption := &thrift.Option{
				Name:  openapiDocumentOption,
				Value: optionStr,
			}
			c.ThriftFile.Services[0].Options = append(c.ThriftFile.Services[0].Options, schemaOption)
			c.AddThriftInclude(openapiThriftFile)
		}
	}

	return nil
}

// Add a new method to handle structured extensions
func (c *ThriftConverter) addExtensionsToProtoOptions() error {
	// Check for x-option in spec extensions
	if xOption, ok := c.spec.Extensions["x-options"]; ok {
		if optionMap, ok := xOption.(map[string]interface{}); ok {
			for key, value := range optionMap {
				c.ThriftFile.Namespace[key] = fmt.Sprintf("%q", value)
			}
		}
	}

	// Check for x-option in spec.info.extensions
	if c.spec.Info != nil {
		if xOption, ok := c.spec.Info.Extensions["x-options"]; ok {
			if optionMap, ok := xOption.(map[string]interface{}); ok {
				for key, value := range optionMap {
					c.ThriftFile.Namespace[key] = fmt.Sprintf("%q", value)
				}
			}
		}
	}

	return nil
}

// convertTagsToThriftServices converts OpenAPI tags into Thrift services and stores them in the ThriftFile
func (c *ThriftConverter) convertTagsToThriftServices() error {
	tags := c.spec.Tags
	for _, tag := range tags {
		serviceName := utils.ToPascaleCase(tag.Name)
		service := &thrift.ThriftService{
			Name:        serviceName,
			Description: tag.Description,
		}
		c.ThriftFile.Services = append(c.ThriftFile.Services, service)
	}
	return nil
}

// convertComponentsToThriftMessages converts OpenAPI components into Thrift messages and stores them in the ThriftFile
func (c *ThriftConverter) convertComponentsToThriftMessages() error {
	components := c.spec.Components
	if components == nil {
		return nil
	}
	if components.Schemas == nil {
		return nil
	}
	for name, schemaRef := range components.Schemas {
		schema := schemaRef
		if c.converterOption.NamingOption {
			name = utils.ToPascaleCase(name)
		}
		thriftType, err := c.ConvertSchemaToThriftType(schema, name, nil)
		if err != nil {
			return fmt.Errorf("error converting schema %s: %w", name, err)
		}
		switch v := thriftType.(type) {
		case *thrift.ThriftField:
			message := &thrift.ThriftStruct{
				Name:   name,
				Fields: []*thrift.ThriftField{v},
			}
			if c.converterOption.OpenapiOption {
				optionStr := utils.StructToOption(schema.Value, "    ")

				schemaOption := &thrift.Option{
					Name:  openapiSchemaOption,
					Value: optionStr,
				}
				message.Options = append(message.Options, schemaOption)
				c.AddThriftInclude(openapiThriftFile)
			}
			c.addMessageToThrift(message)
		case *thrift.ThriftStruct:
			if c.converterOption.OpenapiOption {
				optionStr := utils.StructToOption(schema.Value, "    ")

				schemaOption := &thrift.Option{
					Name:  openapiSchemaOption,
					Value: optionStr,
				}
				v.Options = append(v.Options, schemaOption)
				c.AddThriftInclude(openapiThriftFile)
			}
			c.addMessageToThrift(v)
		case *thrift.ThriftEnum:
			if c.converterOption.OpenapiOption {
				optionStr := utils.StructToOption(schema.Value, "    ")

				schemaOption := &thrift.Option{
					Name:  openapiSchemaOption,
					Value: optionStr,
				}
				v.Options = append(v.Options, schemaOption)
				c.AddThriftInclude(openapiThriftFile)
			}
			c.addEnumToThrift(v)
		case *thrift.ThriftUnion:
			if c.converterOption.OpenapiOption {
				optionStr := utils.StructToOption(schema.Value, "    ")

				schemaOption := &thrift.Option{
					Name:  openapiSchemaOption,
					Value: optionStr,
				}
				v.Options = append(v.Options, schemaOption)
				c.AddThriftInclude(openapiThriftFile)
			}
			c.addUnionToThrift(v)
		}
	}
	return nil
}

// convertPathsToThriftServices converts OpenAPI path items into Thrift services and stores them in the ThriftFile
func (c *ThriftConverter) convertPathsToThriftServices() error {
	paths := c.spec.Paths
	services, err := c.ConvertPathsToThriftServices(paths)
	if err != nil {
		return fmt.Errorf("error converting paths to thrift services: %w", err)
	}

	c.ThriftFile.Services = append(c.ThriftFile.Services, services...)
	return nil
}

// ConvertPathsToThriftServices converts OpenAPI path items into Thrift services
func (c *ThriftConverter) ConvertPathsToThriftServices(paths *openapi3.Paths) ([]*thrift.ThriftService, error) {
	var services []*thrift.ThriftService

	for path, pathItem := range paths.Map() {
		for method, operation := range pathItem.Operations() {
			serviceName := utils.GetServiceName(operation)
			methodName := utils.GetMethodName(operation, path, method)

			if c.converterOption.NamingOption {
				serviceName = utils.ToPascaleCase(serviceName)
				methodName = utils.ToPascaleCase(methodName)
			}

			inputMessage, err := c.generateRequestMessage(operation, methodName)
			if err != nil {
				return nil, fmt.Errorf("error generating request message for %s: %w", methodName, err)
			}

			outputMessage, err := c.generateResponseMessage(operation, methodName)
			if err != nil {
				return nil, fmt.Errorf("error generating response message for %s: %w", methodName, err)
			}

			service := c.findOrCreateService(serviceName)

			if !c.methodExistsInService(service, methodName) {
				thriftMethod := &thrift.ThriftMethod{
					Name:   methodName,
					Input:  inputMessage,
					Output: outputMessage,
				}

				if c.converterOption.ApiOption {
					if optionName, ok := MethodToOption[method]; ok {
						option := &thrift.Option{
							Name:  optionName,
							Value: fmt.Sprintf("%q", utils.ConvertPath(path)),
						}
						thriftMethod.Options = append(thriftMethod.Options, option)
					}
				}

				if c.converterOption.OpenapiOption {
					optionStr := utils.StructToOption(operation, "     ")

					schemaOption := &thrift.Option{
						Name:  "openapi.operation",
						Value: optionStr,
					}
					thriftMethod.Options = append(thriftMethod.Options, schemaOption)
					c.AddThriftInclude(openapiThriftFile)
				}
				service.Methods = append(service.Methods, thriftMethod)
			}
		}
	}

	return services, nil
}

// generateRequestMessage generates a request message for an operation
func (c *ThriftConverter) generateRequestMessage(operation *openapi3.Operation, methodName string) ([]string, error) {
	messageName := utils.GetMessageName(operation, methodName, "Request")

	if c.converterOption.NamingOption {
		messageName = utils.ToPascaleCase(messageName)
	}

	message := &thrift.ThriftStruct{Name: messageName}

	if operation.RequestBody == nil && len(operation.Parameters) == 0 {
		return []string{""}, nil
	}

	if operation.RequestBody != nil {
		if operation.RequestBody.Ref != "" {
			//todo
			return []string{utils.ExtractMessageNameFromRef(operation.RequestBody.Ref)}, nil
		}

		if operation.RequestBody.Value != nil && len(operation.RequestBody.Value.Content) > 0 {
			for mediaTypeStr, mediaType := range operation.RequestBody.Value.Content {
				schema := mediaType.Schema
				if schema != nil {
					thriftType, err := c.ConvertSchemaToThriftType(schema, utils.FormatStr(mediaTypeStr), message)
					if err != nil {
						return []string{""}, err
					}

					switch v := thriftType.(type) {
					case *thrift.ThriftField:
						if c.converterOption.ApiOption {
							var optionName string
							if mediaTypeStr == "application/json" {
								optionName = "api.body"
							} else if mediaTypeStr == "application/x-www-form-urlencoded" || mediaTypeStr == "multipart/form-data" {
								optionName = "api.form"
							}
							if optionName != "" {
								v.Options = append(v.Options, &thrift.Option{
									Name:  optionName,
									Value: fmt.Sprintf("%q", v.Name),
								})
							}
						}
						c.addFieldIfNotExists(&message.Fields, v)
					case *thrift.ThriftStruct:
						for _, field := range v.Fields {
							if c.converterOption.ApiOption {
								var optionName string
								if mediaTypeStr == "application/json" {
									optionName = "api.body"
								} else if mediaTypeStr == "application/x-www-form-urlencoded" || mediaTypeStr == "multipart/form-data" {
									optionName = "api.form"
								}
								if optionName != "" {
									field.Options = append(field.Options, &thrift.Option{
										Name:  optionName,
										Value: fmt.Sprintf("%q", field.Name),
									})
								}
							}
							c.addFieldIfNotExists(&message.Fields, field)
						}
					case *thrift.ThriftEnum:
						name := mediaTypeStr
						if c.converterOption.NamingOption {
							name = utils.ToSnakeCase(name)
						} else {
							name = utils.FormatStr(name)
						}
						newField := &thrift.ThriftField{
							Name: name + "_field",
							Type: v.Name,
						}
						if c.converterOption.ApiOption {
							var optionName string
							if mediaTypeStr == "application/json" {
								optionName = "api.body"
							} else if mediaTypeStr == "application/x-www-form-urlencoded" || mediaTypeStr == "multipart/form-data" {
								optionName = "api.form"
							}
							if optionName != "" {
								newField.Options = append(newField.Options, &thrift.Option{
									Name:  optionName,
									Value: fmt.Sprintf("%q", v.Name),
								})
							}
						}
						if c.converterOption.OpenapiOption {
							optionStr := utils.StructToOption(operation.RequestBody.Value, "     ")

							schemaOption := &thrift.Option{
								Name:  openapiPropertyOption,
								Value: optionStr,
							}
							newField.Options = append(newField.Options, schemaOption)
							c.AddThriftInclude(openapiThriftFile)
						}
						message.Fields = append(message.Fields, newField)
					case *thrift.ThriftUnion:
						name := mediaTypeStr
						if c.converterOption.NamingOption {
							name = utils.ToSnakeCase(name)
						} else {
							name = utils.FormatStr(name)
						}
						newField := &thrift.ThriftField{
							Name: name + "_field",
							Type: v.Name,
						}
						if c.converterOption.ApiOption {
							var optionName string
							if mediaTypeStr == "application/json" {
								optionName = "api.body"
							} else if mediaTypeStr == "application/x-www-form-urlencoded" || mediaTypeStr == "multipart/form-data" {
								optionName = "api.form"
							}
							if optionName != "" {
								newField.Options = append(newField.Options, &thrift.Option{
									Name:  optionName,
									Value: fmt.Sprintf("%q", v.Name),
								})
							}
						}
						if c.converterOption.OpenapiOption {
							optionStr := utils.StructToOption(operation.RequestBody.Value, "     ")

							schemaOption := &thrift.Option{
								Name:  openapiPropertyOption,
								Value: optionStr,
							}
							newField.Options = append(newField.Options, schemaOption)
							c.AddThriftInclude(openapiThriftFile)
						}
						message.Fields = append(message.Fields, newField)
					}
				}
			}
		}
	}

	if len(operation.Parameters) > 0 {
		for _, param := range operation.Parameters {
			if param.Value.Schema != nil {
				fieldOrMessage, err := c.ConvertSchemaToThriftType(param.Value.Schema, param.Value.Name, message)
				if err != nil {
					return []string{""}, err
				}

				switch v := fieldOrMessage.(type) {
				case *thrift.ThriftField:
					if c.converterOption.ApiOption {
						v.Options = append(v.Options, &thrift.Option{
							Name:  "api." + param.Value.In,
							Value: fmt.Sprintf("%q", param.Value.Name),
						})
					}
					if c.converterOption.OpenapiOption {
						optionStr := utils.StructToOption(param.Value, "     ")

						schemaOption := &thrift.Option{
							Name:  openapiParameterOption,
							Value: optionStr,
						}
						v.Options = append(v.Options, schemaOption)
						c.AddThriftInclude(openapiThriftFile)
					}
					v.Description = param.Value.Description
					c.addFieldIfNotExists(&message.Fields, v)
				case *thrift.ThriftStruct:
					for _, field := range v.Fields {
						if c.converterOption.ApiOption {
							field.Options = append(field.Options, &thrift.Option{
								Name:  "api." + param.Value.In,
								Value: fmt.Sprintf("%q", param.Value.Name),
							})
						}
						if c.converterOption.OpenapiOption {
							optionStr := utils.StructToOption(param.Value, "     ")

							schemaOption := &thrift.Option{
								Name:  openapiParameterOption,
								Value: optionStr,
							}
							field.Options = append(field.Options, schemaOption)
							c.AddThriftInclude(openapiThriftFile)
						}
						c.addFieldIfNotExists(&message.Fields, field)
					}
				case *thrift.ThriftEnum:

				case *thrift.ThriftUnion:
					name := param.Value.Name
					if c.converterOption.NamingOption {
						name = utils.ToPascaleCase(name)
					}
					newField := &thrift.ThriftField{
						Name: name,
						Type: v.Name,
					}
					if c.converterOption.ApiOption {
						newField.Options = append(newField.Options, &thrift.Option{
							Name:  "api." + param.Value.In,
							Value: fmt.Sprintf("%q", param.Value.Name),
						})
					}
					if c.converterOption.OpenapiOption {
						optionStr := utils.StructToOption(param.Value, "     ")

						schemaOption := &thrift.Option{
							Name:  openapiParameterOption,
							Value: optionStr,
						}
						newField.Options = append(newField.Options, schemaOption)
						c.AddThriftInclude(openapiThriftFile)
					}
					message.Fields = append(message.Fields, newField)
				}
			}
		}
	}

	// if there are no fields or messages, return an empty message
	if len(message.Fields) > 0 {
		c.addMessageToThrift(message)
		return []string{message.Name}, nil
	}

	return []string{""}, nil
}

// generateResponseMessage generates a response message for an operation
func (c *ThriftConverter) generateResponseMessage(operation *openapi3.Operation, methodName string) (string, error) {
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
		for _, responseRef := range responses {
			if responseRef.Ref == "" && (responseRef.Value == nil || (len(responseRef.Value.Content) == 0 && len(responseRef.Value.Headers) == 0)) {
				continue
			}
			return c.processSingleResponse("", responseRef, operation, methodName)
		}
	}

	if responseCount == 0 {
		return "void", nil
	}

	// create a wrapper message for multiple responses
	wrapperMessageName := utils.GetMessageName(operation, methodName, "Response")
	if c.converterOption.NamingOption {
		wrapperMessageName = utils.ToPascaleCase(wrapperMessageName)
	}

	wrapperMessage := &thrift.ThriftStruct{Name: wrapperMessageName}

	emptyFlag := true

	for statusCode, responseRef := range responses {
		if responseRef.Ref == "" && (responseRef.Value == nil || len(responseRef.Value.Content) == 0) {
			break
		}
		emptyFlag = false
		messageName, err := c.processSingleResponse(statusCode, responseRef, operation, methodName)
		if err != nil {
			return "", err
		}

		name := "Response_" + statusCode
		if c.converterOption.NamingOption {
			name = utils.ToSnakeCase(name)
		}
		field := &thrift.ThriftField{
			Name: name,
			Type: messageName,
		}
		wrapperMessage.Fields = append(wrapperMessage.Fields, field)
	}

	if emptyFlag {
		//c.AddThriftInclude(emptyThriftFile)
		return "void", nil
	}

	c.addMessageToThrift(wrapperMessage)

	return wrapperMessage.Name, nil
}

// processSingleResponse deals with a single response in an operation
func (c *ThriftConverter) processSingleResponse(statusCode string, responseRef *openapi3.ResponseRef, operation *openapi3.Operation, methodName string) (string, error) {
	if responseRef.Ref != "" {
		return utils.ExtractMessageNameFromRef(responseRef.Ref), nil
	}

	response := responseRef.Value
	messageName := operation.OperationID + "Response" + utils.ToUpperCase(statusCode)

	if c.converterOption.NamingOption {
		messageName = utils.ToPascaleCase(messageName)
	}

	message := &thrift.ThriftStruct{Name: messageName}

	if len(response.Headers) > 0 {
		for headerName, headerRef := range response.Headers {
			if headerRef != nil {

				fieldOrMessage, err := c.ConvertSchemaToThriftType(headerRef.Value.Schema, headerName, message)
				if err != nil {
					return "", err
				}

				switch v := fieldOrMessage.(type) {
				case *thrift.ThriftField:
					if c.converterOption.ApiOption {
						option := &thrift.Option{
							Name:  "api.header",
							Value: fmt.Sprintf("%q", headerName),
						}
						v.Options = append(v.Options, option)
					}
					if c.converterOption.OpenapiOption {
						optionStr := utils.StructToOption(headerRef.Value, "     ")

						schemaOption := &thrift.Option{
							Name:  openapiPropertyOption,
							Value: optionStr,
						}
						v.Options = append(v.Options, schemaOption)
						c.AddThriftInclude(openapiThriftFile)
					}
					c.addFieldIfNotExists(&message.Fields, v)
				case *thrift.ThriftStruct:
					for _, field := range v.Fields {
						if c.converterOption.ApiOption {
							option := &thrift.Option{
								Name:  "api.header",
								Value: fmt.Sprintf("%q", field.Name),
							}
							field.Options = append(field.Options, option)
						}
						if c.converterOption.OpenapiOption {
							optionStr := utils.StructToOption(headerRef.Value, "     ")

							schemaOption := &thrift.Option{
								Name:  openapiPropertyOption,
								Value: optionStr,
							}
							field.Options = append(field.Options, schemaOption)
							c.AddThriftInclude(openapiThriftFile)
						}
						c.addFieldIfNotExists(&message.Fields, field)
					}
				case *thrift.ThriftEnum:
					name := headerName
					if c.converterOption.NamingOption {
						name = utils.ToSnakeCase(name)
					}
					newField := &thrift.ThriftField{
						Name: name + "_field",
						Type: v.Name,
					}
					if c.converterOption.ApiOption {
						option := &thrift.Option{
							Name:  "api.header",
							Value: fmt.Sprintf("%q", headerName),
						}
						newField.Options = append(newField.Options, option)
					}
					if c.converterOption.OpenapiOption {
						optionStr := utils.StructToOption(headerRef.Value, "     ")

						schemaOption := &thrift.Option{
							Name:  openapiPropertyOption,
							Value: optionStr,
						}
						newField.Options = append(newField.Options, schemaOption)
						c.AddThriftInclude(openapiThriftFile)
					}
					message.Fields = append(message.Fields, newField)
				case *thrift.ThriftUnion:
					name := headerName
					if c.converterOption.NamingOption {
						name = utils.ToSnakeCase(name)
					}
					newField := &thrift.ThriftField{
						Name: name + "_field",
						Type: v.Name,
					}
					if c.converterOption.ApiOption {
						option := &thrift.Option{
							Name:  "api.header",
							Value: fmt.Sprintf("%q", headerName),
						}
						newField.Options = append(newField.Options, option)
					}
					if c.converterOption.OpenapiOption {
						optionStr := utils.StructToOption(headerRef.Value, "     ")

						schemaOption := &thrift.Option{
							Name:  openapiPropertyOption,
							Value: optionStr,
						}
						newField.Options = append(newField.Options, schemaOption)
						c.AddThriftInclude(openapiThriftFile)
					}
					message.Fields = append(message.Fields, newField)
				}
			}
		}
	}

	for mediaTypeStr, mediaType := range response.Content {
		schema := mediaType.Schema
		if schema != nil {

			thriftType, err := c.ConvertSchemaToThriftType(schema, utils.FormatStr(mediaTypeStr), message)
			if err != nil {
				return "", err
			}

			switch v := thriftType.(type) {
			case *thrift.ThriftField:
				if c.converterOption.ApiOption && mediaTypeStr == "application/json" {
					option := &thrift.Option{
						Name:  "api.body",
						Value: fmt.Sprintf("%q", v.Name),
					}
					v.Options = append(v.Options, option)
				}
				c.addFieldIfNotExists(&message.Fields, v)
			case *thrift.ThriftStruct:
				for _, field := range v.Fields {
					if c.converterOption.ApiOption && mediaTypeStr == "application/json" {
						option := &thrift.Option{
							Name:  "api.body",
							Value: fmt.Sprintf("%q", field.Name),
						}
						field.Options = append(field.Options, option)
					}
					if c.converterOption.OpenapiOption {
						optionStr := utils.StructToOption(schema, "     ")

						schemaOption := &thrift.Option{
							Name:  openapiPropertyOption,
							Value: optionStr,
						}
						field.Options = append(field.Options, schemaOption)
						c.AddThriftInclude(openapiThriftFile)
					}
					c.addFieldIfNotExists(&message.Fields, field)
				}
			case *thrift.ThriftEnum:
				name := mediaTypeStr
				if c.converterOption.NamingOption {
					name = utils.ToSnakeCase(mediaTypeStr)
				} else {
					name = utils.ToUpperCase(name)
				}
				newField := &thrift.ThriftField{
					Name: name,
					Type: v.Name,
				}
				if c.converterOption.ApiOption && mediaTypeStr == "application/json" {
					option := &thrift.Option{
						Name:  "api.body",
						Value: fmt.Sprintf("%q", v.Name),
					}
					newField.Options = append(newField.Options, option)
				}
				if c.converterOption.OpenapiOption {
					optionStr := utils.StructToOption(schema, "     ")

					schemaOption := &thrift.Option{
						Name:  openapiPropertyOption,
						Value: optionStr,
					}
					newField.Options = append(newField.Options, schemaOption)
					c.AddThriftInclude(openapiThriftFile)
				}
				message.Fields = append(message.Fields, newField)
			case *thrift.ThriftUnion:
				name := mediaTypeStr
				if c.converterOption.NamingOption {
					name = utils.ToSnakeCase(mediaTypeStr)
				} else {
					name = utils.ToUpperCase(name)
				}
				newField := &thrift.ThriftField{
					Name: name,
					Type: v.Name,
				}
				if c.converterOption.ApiOption && mediaTypeStr == "application/json" {
					option := &thrift.Option{
						Name:  "api.body",
						Value: fmt.Sprintf("%q", v.Name),
					}
					newField.Options = append(newField.Options, option)
				}
				if c.converterOption.OpenapiOption {
					optionStr := utils.StructToOption(schema, "     ")

					schemaOption := &thrift.Option{
						Name:  openapiPropertyOption,
						Value: optionStr,
					}
					newField.Options = append(newField.Options, schemaOption)
					c.AddThriftInclude(openapiThriftFile)
				}
				message.Fields = append(message.Fields, newField)
			}
		}
	}

	if len(message.Fields) > 0 {
		c.addMessageToThrift(message)
		return message.Name, nil
	}
	return "", nil
}

// ConvertSchemaToThriftType converts an OpenAPI schema to a Thrift field or message
func (c *ThriftConverter) ConvertSchemaToThriftType(
	schemaRef *openapi3.SchemaRef,
	thriftName string,
	parentMessage *thrift.ThriftStruct) (interface{}, error) {

	var thriftType string
	var result interface{}

	// Handle referenced schema
	if schemaRef.Ref != "" {
		name := c.applyNamingOption(utils.ExtractMessageNameFromRef(schemaRef.Ref))
		return &thrift.ThriftField{
			Name: name,
			Type: utils.ExtractMessageNameFromRef(schemaRef.Ref),
		}, nil
	}

	// Ensure schema value is valid
	if schemaRef.Value == nil {
		return nil, errors.New("schema type is required")
	}

	schema := schemaRef.Value
	description := schema.Description

	// Handle oneOf, allOf, anyOf even if schema.Type is nil
	if len(schema.OneOf) > 0 {
		thriftStruct, err := c.handleOneOf(schema.OneOf, thriftName, parentMessage)
		if err != nil {
			return nil, err
		}
		return thriftStruct, nil
	} else if len(schema.AllOf) > 0 {
		thriftStruct, err := c.handleAllOf(schema.AllOf, thriftName, parentMessage)
		if err != nil {
			return nil, err
		}
		return thriftStruct, nil
	} else if len(schema.AnyOf) > 0 {
		thriftStruct, err := c.handleAnyOf(schema.AnyOf, thriftName, parentMessage)
		if err != nil {
			return nil, err
		}
		return thriftStruct, nil
	}

	// Process schema type
	switch {
	case schema.Type.Includes("string"):
		if schema.Format == "date" || schema.Format == "date-time" {
			thriftType = "string"
		} else if schema.Format == "byte" || schema.Format == "binary" {
			thriftType = "binary"
		} else if len(schema.Enum) != 0 {
			var name string
			if parentMessage == nil {
				name = thriftName
			} else {
				name = c.applyNamingOption(utils.ToUpperCase(thriftName))
			}
			thriftEnum := &thrift.ThriftEnum{
				Name:        name + "Enum",
				Description: description,
			}
			for i, enumValue := range schema.Enum {
				thriftEnum.Values = append(thriftEnum.Values, &thrift.ThriftEnumValue{
					Index: i,
					Value: enumValue,
				})
			}
			result = thriftEnum
		} else {
			thriftType = "string"
		}

	case schema.Type.Includes("integer"):
		if len(schema.Enum) != 0 {
			var name string
			if parentMessage == nil {
				name = thriftName
			} else {
				name = c.applyNamingOption(utils.ToUpperCase(thriftName))
			}
			thriftEnum := &thrift.ThriftEnum{
				Name:        name + "Enum",
				Description: description,
			}
			for i, enumValue := range schema.Enum {
				thriftEnum.Values = append(thriftEnum.Values, &thrift.ThriftEnumValue{
					Index: i,
					Value: enumValue,
				})
			}
			result = thriftEnum
		} else if schema.Format == "int32" {
			thriftType = "i32"
		} else {
			thriftType = "i64"
		}

	case schema.Type.Includes("number"):
		if len(schema.Enum) != 0 {
			var name string
			if parentMessage == nil {
				name = thriftName
			} else {
				name = c.applyNamingOption(utils.ToUpperCase(thriftName))
			}
			thriftEnum := &thrift.ThriftEnum{
				Name:        name + "Enum",
				Description: description,
			}
			for i, enumValue := range schema.Enum {
				thriftEnum.Values = append(thriftEnum.Values, &thrift.ThriftEnumValue{
					Index: i,
					Value: enumValue,
				})
			}
			result = thriftEnum
		} else if schema.Format == "float" {
			thriftType = "float"
		} else {
			thriftType = "double"
		}

	case schema.Type.Includes("boolean"):
		thriftType = "bool"

	case schema.Type.Includes("array"):
		if schema.Items != nil {
			fieldOrMessage, err := c.ConvertSchemaToThriftType(schema.Items, thriftName+"Item", parentMessage)
			if err != nil {
				return nil, err
			}

			fieldType := ""
			if field, ok := fieldOrMessage.(*thrift.ThriftField); ok {
				fieldType = field.Type
			} else if nestedMessage, ok := fieldOrMessage.(*thrift.ThriftStruct); ok {
				fieldType = nestedMessage.Name
				c.addMessageToThrift(nestedMessage)
			} else if enum, ok := fieldOrMessage.(*thrift.ThriftEnum); ok {
				fieldType = enum.Name
				c.addEnumToThrift(enum)
			} else if union, ok := fieldOrMessage.(*thrift.ThriftUnion); ok {
				fieldType = union.Name
				c.addUnionToThrift(union)
			}

			result = &thrift.ThriftField{
				Name:        c.applyNamingOption(thriftName),
				Type:        fieldType,
				Repeated:    true,
				Description: description,
			}
		}

	case schema.Type.Includes("object"):

		// Regular object handling
		var message *thrift.ThriftStruct
		if parentMessage == nil {
			message = &thrift.ThriftStruct{Name: thriftName}
		} else {
			message = &thrift.ThriftStruct{Name: c.applyNamingOption(utils.ToUpperCase(thriftName))}
		}

		// Process each property in the object
		for propName, propSchema := range schema.Properties {
			thriftType, err := c.ConvertSchemaToThriftType(propSchema, propName, message)
			if err != nil {
				return nil, err
			}

			// Add the converted fields to the message
			if field, ok := thriftType.(*thrift.ThriftField); ok {
				if c.converterOption.OpenapiOption {
					optionStr := utils.StructToOption(propSchema, "    ")

					schemaOption := &thrift.Option{
						Name:  openapiPropertyOption,
						Value: optionStr,
					}
					field.Options = append(field.Options, schemaOption)
					c.AddThriftInclude(openapiThriftFile)
				}
				message.Fields = append(message.Fields, field)
			} else if nestedMessage, ok := thriftType.(*thrift.ThriftStruct); ok {
				var name string
				if c.converterOption.NamingOption {
					name = utils.ToSnakeCase(nestedMessage.Name)
				}
				newField := &thrift.ThriftField{
					Name: name + "_field",
					Type: nestedMessage.Name,
				}
				if c.converterOption.OpenapiOption {
					optionStr := utils.StructToOption(propSchema.Value, "    ")

					schemaOption := &thrift.Option{
						Name:  openapiPropertyOption,
						Value: optionStr,
					}
					newField.Options = append(newField.Options, schemaOption)
					c.AddThriftInclude(openapiThriftFile)
				}
				c.addMessageToThrift(nestedMessage)
				message.Fields = append(message.Fields, newField)
			} else if enum, ok := thriftType.(*thrift.ThriftEnum); ok {
				c.addEnumToThrift(enum)
				message.Fields = append(message.Fields, &thrift.ThriftField{
					Name: c.applyNamingOption(propName),
					Type: enum.Name,
				})
			} else if union, ok := thriftType.(*thrift.ThriftUnion); ok {
				c.addUnionToThrift(union)
				message.Fields = append(message.Fields, &thrift.ThriftField{
					Name: c.applyNamingOption(propName),
					Type: union.Name,
				})
			}
		}

		// Handle additionalProperties if present
		if schema.AdditionalProperties.Schema != nil {
			mapValueType := "string"
			additionalPropMessage, err := c.ConvertSchemaToThriftType(schema.AdditionalProperties.Schema, thriftName+"AdditionalProperties", parentMessage)
			if err != nil {
				return nil, err
			}
			if msg, ok := additionalPropMessage.(*thrift.ThriftStruct); ok {
				mapValueType = msg.Name
			} else if enum, ok := additionalPropMessage.(*thrift.ThriftEnum); ok {
				mapValueType = enum.Name
			}

			message.Fields = append(message.Fields, &thrift.ThriftField{
				Name: "additionalProperties",
				Type: "map<string, " + mapValueType + ">",
			})
		}

		// Set the result as the final message
		message.Description = description
		result = message
	}

	// If result is still nil, construct a default ThriftField
	if result == nil {
		result = &thrift.ThriftField{
			Name:        c.applyNamingOption(thriftName),
			Type:        thriftType,
			Description: description,
		}
	}

	return result, nil
}

func (c *ThriftConverter) handleOneOf(oneOfSchemas []*openapi3.SchemaRef, thriftName string, parentMessage *thrift.ThriftStruct) (*thrift.ThriftUnion, error) {
	oneOfUnion := &thrift.ThriftUnion{
		Name: thriftName + "OneOf",
	}

	for i, schemaRef := range oneOfSchemas {
		fieldName := fmt.Sprintf("%sOption%d", thriftName, i+1)
		thriftType, err := c.ConvertSchemaToThriftType(schemaRef, fieldName, parentMessage)
		if err != nil {
			return nil, err
		}

		switch v := thriftType.(type) {
		case *thrift.ThriftField:
			oneOfUnion.Fields = append(oneOfUnion.Fields, v)
		case *thrift.ThriftStruct:
			newField := &thrift.ThriftField{
				Name: v.Name + "_field",
				Type: v.Name,
			}
			c.addMessageToThrift(v)
			oneOfUnion.Fields = append(oneOfUnion.Fields, newField)
		case *thrift.ThriftEnum:
			newField := &thrift.ThriftField{
				Name: v.Name + "_field",
				Type: v.Name,
			}
			c.addEnumToThrift(v)
			oneOfUnion.Fields = append(oneOfUnion.Fields, newField)
		case *thrift.ThriftUnion:
			newField := &thrift.ThriftField{
				Name: v.Name + "_field",
				Type: v.Name,
			}
			c.addUnionToThrift(v)
			oneOfUnion.Fields = append(oneOfUnion.Fields, newField)
		}
	}

	return oneOfUnion, nil
}

func (c *ThriftConverter) handleAllOf(allOfSchemas []*openapi3.SchemaRef, thriftName string, parentMessage *thrift.ThriftStruct) (*thrift.ThriftStruct, error) {
	allOfStruct := &thrift.ThriftStruct{
		Name: thriftName + "AllOf",
	}

	for i, schemaRef := range allOfSchemas {
		fieldName := fmt.Sprintf("%sPart%d", thriftName, i+1)
		thriftType, err := c.ConvertSchemaToThriftType(schemaRef, fieldName, parentMessage)
		if err != nil {
			return nil, err
		}

		switch v := thriftType.(type) {
		case *thrift.ThriftField:
			allOfStruct.Fields = append(allOfStruct.Fields, v)
		case *thrift.ThriftStruct:
			newField := &thrift.ThriftField{
				Name: v.Name + "_field",
				Type: v.Name,
			}
			c.addMessageToThrift(v)
			allOfStruct.Fields = append(allOfStruct.Fields, newField)
		case *thrift.ThriftEnum:
			newField := &thrift.ThriftField{
				Name: v.Name + "_field",
				Type: v.Name,
			}
			c.addEnumToThrift(v)
			allOfStruct.Fields = append(allOfStruct.Fields, newField)
		case *thrift.ThriftUnion:
			newField := &thrift.ThriftField{
				Name: v.Name + "_field",
				Type: v.Name,
			}
			c.addUnionToThrift(v)
			allOfStruct.Fields = append(allOfStruct.Fields, newField)
		}
	}

	return allOfStruct, nil
}

func (c *ThriftConverter) handleAnyOf(anyOfSchemas []*openapi3.SchemaRef, thriftName string, parentMessage *thrift.ThriftStruct) (*thrift.ThriftStruct, error) {
	anyOfStruct := &thrift.ThriftStruct{
		Name: thriftName + "AnyOf",
	}

	for i, schemaRef := range anyOfSchemas {
		fieldName := fmt.Sprintf("%sOption%d", thriftName, i+1)
		thriftType, err := c.ConvertSchemaToThriftType(schemaRef, fieldName, parentMessage)
		if err != nil {
			return nil, err
		}

		switch v := thriftType.(type) {
		case *thrift.ThriftField:
			anyOfStruct.Fields = append(anyOfStruct.Fields, v)
		case *thrift.ThriftStruct:
			newField := &thrift.ThriftField{
				Name: v.Name + "_field",
				Type: v.Name,
			}
			c.addMessageToThrift(v)
			anyOfStruct.Fields = append(anyOfStruct.Fields, newField)
		case *thrift.ThriftEnum:
			newField := &thrift.ThriftField{
				Name: v.Name + "_field",
				Type: v.Name,
			}
			c.addEnumToThrift(v)
			anyOfStruct.Fields = append(anyOfStruct.Fields, newField)
		case *thrift.ThriftUnion:
			newField := &thrift.ThriftField{
				Name: v.Name + "_field",
				Type: v.Name,
			}
			c.addUnionToThrift(v)
			anyOfStruct.Fields = append(anyOfStruct.Fields, newField)
		}
	}

	return anyOfStruct, nil
}

// applyNamingOption applies naming convention based on the converter's naming option
func (c *ThriftConverter) applyNamingOption(name string) string {
	if c.converterOption.NamingOption {
		return utils.ToPascaleCase(name)
	}
	return name
}

// addMessageToThrift adds a ThriftStruct to the ThriftFile globally
func (c *ThriftConverter) addMessageToThrift(message *thrift.ThriftStruct) error {
	if message == nil {
		return errors.New("message is nil")
	}

	// Check if the message already exists in the ThriftFile
	for _, existingMessage := range c.ThriftFile.Structs {
		if existingMessage.Name == message.Name {
			// Merge fields if the message already exists
			fieldNames := make(map[string]struct{})
			for _, field := range existingMessage.Fields {
				fieldNames[field.Name] = struct{}{}
			}
			for _, newField := range message.Fields {
				if _, exists := fieldNames[newField.Name]; !exists {
					existingMessage.Fields = append(existingMessage.Fields, newField)
				}
			}
			return nil
		}
	}

	// Add the message globally
	c.ThriftFile.Structs = append(c.ThriftFile.Structs, message)
	return nil
}

// addEnumToThrift adds an enum to the ThriftFile
func (c *ThriftConverter) addEnumToThrift(enum *thrift.ThriftEnum) {
	c.ThriftFile.Enums = append(c.ThriftFile.Enums, enum)
}

// addUnionToThrift adds a union to the ThriftFile
func (c *ThriftConverter) addUnionToThrift(union *thrift.ThriftUnion) {
	c.ThriftFile.Unions = append(c.ThriftFile.Unions, union)
}

// AddThriftInclude adds an include to the ThriftFile
func (c *ThriftConverter) AddThriftInclude(includeFile string) {
	if c.ThriftFile != nil {
		for _, existingInclude := range c.ThriftFile.Includes {
			if existingInclude == includeFile {
				return
			}
		}
		c.ThriftFile.Includes = append(c.ThriftFile.Includes, includeFile)
	}
}

// addFieldIfNotExists adds a field to Fields if it does not already exist
func (c *ThriftConverter) addFieldIfNotExists(fields *[]*thrift.ThriftField, field *thrift.ThriftField) {
	for _, existingField := range *fields {
		if existingField.Name == field.Name {
			return
		}
	}
	*fields = append(*fields, field)
}

// addMessageIfNotExists adds a message to Messages if it does not already exist
func (c *ThriftConverter) addMessageIfNotExists(messages *[]*thrift.ThriftStruct, nestedMessage *thrift.ThriftStruct) {
	for _, existingMessage := range *messages {
		if existingMessage.Name == nestedMessage.Name {
			return
		}
	}
	*messages = append(*messages, nestedMessage)
}

// methodExistsInService checks if a method exists in a service
func (c *ThriftConverter) methodExistsInService(service *thrift.ThriftService, methodName string) bool {
	for _, method := range service.Methods {
		if method.Name == methodName {
			return true
		}
	}
	return false
}

// findOrCreateService finds or creates a service
func (c *ThriftConverter) findOrCreateService(serviceName string) *thrift.ThriftService {
	for i := range c.ThriftFile.Services {
		if c.ThriftFile.Services[i].Name == serviceName {
			return c.ThriftFile.Services[i]
		}
	}

	// If no existing service is found, create a new one
	newService := &thrift.ThriftService{Name: serviceName}
	c.ThriftFile.Services = append(c.ThriftFile.Services, newService)
	return newService
}
