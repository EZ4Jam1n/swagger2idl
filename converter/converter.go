package converter

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/swagger-generate/swagger2idl/protobuf"
)

// ProtoConverter 结构体，用于转换 OpenAPI 规范为 Proto 文件
type ProtoConverter struct {
	ProtoFile *protobuf.ProtoFile
}

// NewProtoConverter 创建并初始化 ProtoConverter
func NewProtoConverter(packageName string) *ProtoConverter {
	return &ProtoConverter{
		ProtoFile: &protobuf.ProtoFile{
			PackageName: packageName,
			Messages:    []*protobuf.ProtoMessage{},
			Services:    []protobuf.ProtoService{},
		},
	}
}

// Convert 将 OpenAPI 规范转换为 Proto 文件
func (c *ProtoConverter) Convert(spec *openapi3.T) error {
	// 转换 components 为 Proto 消息
	err := c.convertComponentsToProtoMessages(spec.Components)
	if err != nil {
		return fmt.Errorf("error converting components to proto messages: %w", err)
	}

	// 转换 paths 为 Proto 服务
	err = c.convertPathsToProtoServices(spec.Paths)
	if err != nil {
		return fmt.Errorf("error converting paths to proto services: %w", err)
	}

	return nil
}

// convertComponentsToProtoMessages 转换 OpenAPI components 为 Proto 消息并存入 ProtoFile
func (c *ProtoConverter) convertComponentsToProtoMessages(components *openapi3.Components) error {
	for name, schemaRef := range components.Schemas {
		schema := schemaRef.Value
		message, err := c.ConvertToProtoMessage(schema, name)
		if err != nil {
			return fmt.Errorf("error converting schema %s: %w", name, err)
		}
		// 将生成的 ProtoMessage 添加到 ProtoFile 中
		c.addMessageToProto(message)
	}
	return nil
}

// convertPathsToProtoServices 转换 OpenAPI 的路径项为 Proto 服务并存入 ProtoFile
func (c *ProtoConverter) convertPathsToProtoServices(paths *openapi3.Paths) error {
	services, err := c.ConvertPathsToProtoServices(paths)
	if err != nil {
		return fmt.Errorf("error converting paths to proto services: %w", err)
	}

	c.ProtoFile.Services = append(c.ProtoFile.Services, services...)
	return nil
}

// ConvertToProtoMessage 将 OpenAPI Schema 转换为 Protobuf 消息
func (c *ProtoConverter) ConvertToProtoMessage(schema *openapi3.Schema, name string) (*protobuf.ProtoMessage, error) {
	// 创建新的 ProtoMessage
	protoMsg := &protobuf.ProtoMessage{
		Name: name,
	}

	// 无论是否有 additionalProperties，只要是 object 类型，都生成 message
	if schema.Type != nil && schema.Type.Includes("object") {
		// 处理 object 类型的属性
		for propName, propSchema := range schema.Properties {
			field, err := convertSchemaToProtoField(propName, propSchema.Value)
			if err != nil {
				return nil, err
			}
			protoMsg.Fields = append(protoMsg.Fields, *field)
		}

		// 处理 additionalProperties，如果有的话
		if schema.AdditionalProperties.Schema != nil {
			nestedField, err := c.handleNestedAdditionalProperties(schema, name+"Nested")
			if err != nil {
				return nil, err
			}
			protoMsg.Fields = append(protoMsg.Fields, nestedField)
		}
	} else {
		// 处理非 object 类型
		mainType, err := getPrimaryType(schema.Type)
		if err != nil {
			return nil, err
		}

		switch mainType {
		case "string", "integer", "boolean":
			// 生成单一字段消息
			field := protobuf.ProtoField{
				Name: "value",
				Type: convertSimpleTypeToProto(mainType),
			}
			protoMsg.Fields = append(protoMsg.Fields, field)

		case "number":
			// 处理 number 类型的格式
			format := schema.Format
			var protoType string
			if format == "float" {
				protoType = "float"
			} else if format == "double" {
				protoType = "double"
			} else {
				return nil, fmt.Errorf("unsupported number format: %s", format)
			}
			field := protobuf.ProtoField{
				Name: "value",
				Type: protoType,
			}
			protoMsg.Fields = append(protoMsg.Fields, field)

		case "array":
			// 处理数组类型
			if schema.Items != nil && schema.Items.Value != nil {
				itemType := convertSimpleTypeToProto(getFirstType(schema.Items.Value.Type))
				field := protobuf.ProtoField{
					Name: "items",
					Type: "repeated " + itemType,
				}
				protoMsg.Fields = append(protoMsg.Fields, field)
			} else {
				return nil, fmt.Errorf("array type schema must have items defined")
			}

		default:
			return nil, fmt.Errorf("unsupported root schema type: %s", mainType)
		}
	}

	return protoMsg, nil
}

// ConvertPathsToProtoServices 将 OpenAPI 的路径项转换为 Proto 服务
func (c *ProtoConverter) ConvertPathsToProtoServices(paths *openapi3.Paths) ([]protobuf.ProtoService, error) {
	var services []protobuf.ProtoService

	for _, pathItem := range paths.Map() {
		for method, operation := range pathItem.Operations() {
			// 根据 operation.Tags 或其他逻辑生成服务名称
			serviceName := getServiceName(operation.Tags)

			// 生成方法名，通常基于 operationID 或 HTTP 方法
			methodName := generateMethodName(operation.OperationID, method)

			// 生成输入消息（request message）
			inputMessage, err := c.generateRequestMessage(operation)
			if err != nil {
				return nil, fmt.Errorf("error generating request message for %s: %w", methodName, err)
			}

			// 生成输出消息（response message）
			outputMessage, err := c.generateResponseMessage(operation)
			if err != nil {
				return nil, fmt.Errorf("error generating response message for %s: %w", methodName, err)
			}

			// 查找或创建服务
			service := findOrCreateService(&services, serviceName)

			// 检查当前方法是否已经存在于该服务中，避免重复添加
			if !methodExistsInService(service, methodName) {
				// 将方法添加到服务中
				service.Methods = append(service.Methods, protobuf.ProtoMethod{
					Name:   methodName,
					Input:  inputMessage,
					Output: outputMessage,
				})
			}
		}
	}

	return services, nil
}

// findOrCreateService 查找服务，如果不存在则创建新服务
func findOrCreateService(services *[]protobuf.ProtoService, serviceName string) *protobuf.ProtoService {
	for i := range *services {
		if (*services)[i].Name == serviceName {
			return &(*services)[i]
		}
	}

	// 如果未找到，创建新服务
	newService := protobuf.ProtoService{Name: serviceName}
	*services = append(*services, newService)
	return &(*services)[len(*services)-1]
}

// methodExistsInService 检查方法是否已经存在于服务中，避免重复添加
func methodExistsInService(service *protobuf.ProtoService, methodName string) bool {
	for _, method := range service.Methods {
		if method.Name == methodName {
			return true
		}
	}
	return false
}

func (c *ProtoConverter) handleNestedAdditionalProperties(schema *openapi3.Schema, defaultName string) (protobuf.ProtoField, error) {
	if schema.AdditionalProperties.Schema != nil {
		// 处理嵌套的 additionalProperties
		keyType := "string" // Map 的键通常是字符串
		var valueType string
		if schema.AdditionalProperties.Schema.Value.Type.Includes("object") {
			// 如果值类型是 object，递归处理
			nestedMsg, err := c.ConvertToProtoMessage(schema.AdditionalProperties.Schema.Value, schema.Title)
			if err != nil {
				return protobuf.ProtoField{}, err
			}
			valueType = nestedMsg.Name
		} else {
			// 处理简单类型
			valueType = convertSimpleTypeToProto(getFirstType(schema.AdditionalProperties.Schema.Value.Type))
		}

		// 如果 schema.Title 为空，使用默认名称
		fieldName := schema.Title
		if fieldName == "" {
			fieldName = defaultName // 使用提供的默认名称
		}

		field := protobuf.ProtoField{
			Name: fieldName, // 使用 schema 的标题作为字段名或默认名称
			Type: fmt.Sprintf("map<%s, %s>", keyType, valueType),
		}
		return field, nil
	}
	return protobuf.ProtoField{}, fmt.Errorf("additionalProperties must have a schema")
}

// generateMethodName 生成方法名，基于 OperationID 和 HTTP 方法
func generateMethodName(operationID, method string) string {
	if operationID != "" {
		return operationID
	}
	// 如果没有 OperationID，用 HTTP 方法生成
	return strings.Title(strings.ToLower(method)) + "Method"
}

// generateRequestMessage 生成请求消息
func (c *ProtoConverter) generateRequestMessage(operation *openapi3.Operation) (string, error) {
	messageName := operation.OperationID + "Request"
	message := &protobuf.ProtoMessage{Name: messageName}

	// 处理 requestBody
	if operation.RequestBody != nil {
		// 如果是 $ref，直接返回引用的消息名
		if operation.RequestBody.Ref != "" {
			return extractMessageNameFromRef(operation.RequestBody.Ref), nil
		}

		if operation.RequestBody.Value != nil {
			for _, mediaType := range operation.RequestBody.Value.Content {
				schema := mediaType.Schema
				if schema != nil && schema.Value != nil {
					if schema.Ref != "" {
						return extractMessageNameFromRef(schema.Ref), nil
					}
					// 生成消息字段并加入 message
					requestMsg, err := c.ConvertToProtoMessage(schema.Value, messageName)
					if err != nil {
						return "", err
					}
					// 把 requestBody 的消息字段添加到 message 中
					message.Fields = append(message.Fields, requestMsg.Fields...)
				}
			}
		}

	}

	// 处理 Parameters（query, path, header 等）
	if len(operation.Parameters) > 0 {
		for _, param := range operation.Parameters {
			if param.Value.Schema != nil && param.Value.Schema.Value != nil {
				var field protobuf.ProtoField

				// 动态判断参数类型
				schemaType := getFirstType(param.Value.Schema.Value.Type)

				switch param.Value.In {
				case "path", "cookie", "header", "query":
					// 动态根据不同类型来生成对应的Proto类型
					switch schemaType {
					case "string":
						fieldType := "string"
						field = protobuf.ProtoField{
							Name: param.Value.Name,
							Type: fieldType,
						}
					case "number", "integer":
						fieldType := "int32" // 或 "int64" 视情况而定
						field = protobuf.ProtoField{
							Name: param.Value.Name,
							Type: fieldType,
						}
					case "boolean":
						fieldType := "bool"
						field = protobuf.ProtoField{
							Name: param.Value.Name,
							Type: fieldType,
						}
					case "array":
						// 假设数组元素是简单类型，获取items的类型并处理
						arrayType := convertSimpleTypeToProto(getFirstType(param.Value.Schema.Value.Items.Value.Type))
						fieldType := "repeated " + arrayType
						field = protobuf.ProtoField{
							Name: param.Value.Name,
							Type: fieldType,
						}
					case "object":
						// 如果是对象类型，需要递归生成子消息
						objectMessage, err := c.ConvertToProtoMessage(param.Value.Schema.Value, param.Value.Name)
						if err != nil {
							return "", err
						}
						c.addMessageToProto(objectMessage)

						field = protobuf.ProtoField{
							Name: param.Value.Name,
							Type: objectMessage.Name,
						}
					default:
						// 忽略未知的类型
						continue
					}
				default:
					// 忽略不支持的参数类型
					continue
				}

				// 将生成的字段加入当前的 Protobuf 消息
				message.Fields = append(message.Fields, field)
			}
		}
	}

	// 如果生成了任何字段，添加 message 到 ProtoFile 并返回消息名
	if len(message.Fields) > 0 {
		c.addMessageToProto(message)
		return message.Name, nil
	}

	// 如果没有参数或 requestBody，返回空
	return "", nil
}

// generateResponseMessage 生成响应消息
func (c *ProtoConverter) generateResponseMessage(operation *openapi3.Operation) (string, error) {
	if operation.Responses != nil {
		for statusCode, responseRef := range operation.Responses.Map() {
			// 如果是 $ref，直接返回引用的消息名
			if responseRef.Ref != "" {
				return extractMessageNameFromRef(responseRef.Ref), nil
			}

			response := responseRef.Value
			if response != nil && len(response.Content) > 0 {
				for _, mediaType := range response.Content {
					schema := mediaType.Schema
					if schema != nil && schema.Value != nil {
						if schema.Ref != "" {
							return extractMessageNameFromRef(schema.Ref), nil
						}
						messageName := operation.OperationID + "Response" + statusCode
						// 生成消息字段并加入 message
						responseMsg, err := c.ConvertToProtoMessage(schema.Value, messageName)
						if err != nil {
							return "", err
						}
						// 把 requestBody 的消息字段添加到 message 中
						c.addMessageToProto(responseMsg)
						return responseMsg.Name, nil
					}
					//schema := mediaType.Schema
					//if schema != nil && schema.Value != nil {
					//	messageName := operation.OperationID + "Response" + statusCode
					//
					//	if schema.Ref != "" {
					//		// 如果是引用，返回引用名
					//		return extractMessageNameFromRef(schema.Ref), nil
					//	}
					//
					//	// 处理对象类型
					//	if schema.Value.Type != nil && len(*schema.Value.Type) > 0 {
					//		mainType, err := getPrimaryType(schema.Value.Type)
					//		if err != nil {
					//			return "", err
					//		}
					//
					//		if mainType == "object" {
					//			// 创建一个新的 Protobuf 消息
					//			message := &protobuf.ProtoMessage{Name: messageName}
					//
					//			// 遍历 properties 并生成 Protobuf 字段
					//			for propName, propSchemaRef := range schema.Value.Properties {
					//				propSchema := propSchemaRef.Value
					//				if propSchema != nil {
					//					field, err := convertSchemaToProtoField(propName, propSchema)
					//					if err != nil {
					//						return "", err
					//					}
					//					message.Fields = append(message.Fields, *field)
					//				}
					//			}
					//
					//			// 添加生成的消息到 ProtoFile
					//			c.addMessageToProto(message)
					//			return message.Name, nil
					//		}
					//
					//		// 处理简单类型 (string, integer, boolean) 等其他类型
					//		if mainType == "string" || mainType == "integer" || mainType == "boolean" {
					//			message := &protobuf.ProtoMessage{
					//				Name: messageName,
					//			}
					//
					//			// 创建单一字段的响应消息
					//			field := protobuf.ProtoField{
					//				Name: "value",
					//				Type: convertSimpleTypeToProto(mainType),
					//			}
					//			message.Fields = append(message.Fields, field)
					//
					//			c.addMessageToProto(message)
					//			return message.Name, nil
					//		}
					//	}
					//
					//	// 处理复杂类型
					//	message, err := c.ConvertToProtoMessage(schema.Value, messageName)
					//	if err != nil {
					//		return "", err
					//	}
					//
					//	c.addMessageToProto(message)
					//	return message.Name, nil
					//}
				}
			}
		}
	}

	return "", nil
}

// convertSchemaToProtoField 根据 OpenAPI schema 生成 Protobuf 字段，递归处理嵌套类型
func convertSchemaToProtoField(fieldName string, schema *openapi3.Schema) (*protobuf.ProtoField, error) {
	var protoType string

	// 检查 schema 是否有组合类型定义 (oneOf, anyOf, allOf)
	if len(schema.OneOf) > 0 {
		// 处理 oneOf 组合类型，生成 Protobuf 的 oneof
		return handleOneOf(fieldName, schema.OneOf)
	}

	if len(schema.AnyOf) > 0 {
		// 处理 anyOf 组合类型，生成 Protobuf 的 oneof（语义上与 oneOf 类似）
		return handleOneOf(fieldName, schema.AnyOf) // 与 oneOf 逻辑相似
	}

	if len(schema.AllOf) > 0 {
		// 处理 allOf 组合类型，将所有字段合并
		return handleAllOf(fieldName, schema.AllOf)
	}

	if len(schema.Type.Slice()) > 0 {
		mainType, err := getPrimaryType(schema.Type)
		if err != nil {
			return nil, err
		}

		// 根据类型处理不同情况
		switch mainType {
		case "string":
			protoType = "string"

		case "integer":
			// 检查是否有 format (int32 或 int64)
			if schema.Format == "int64" {
				protoType = "int64"
			} else {
				protoType = "int32" // 默认为 int32
			}

		case "boolean":
			protoType = "bool"

		case "number":
			// 处理 number 类型 (float, double)
			if schema.Format == "float" {
				protoType = "float"
			} else if schema.Format == "double" {
				protoType = "double"
			} else {
				return nil, fmt.Errorf("unsupported number format: %s", schema.Format)
			}

		case "array":
			// 处理数组类型：数组字段类型应该是 repeated
			if schema.Items != nil && schema.Items.Value != nil {
				itemType := convertSimpleTypeToProto(getFirstType(schema.Items.Value.Type))
				protoType = "repeated " + itemType
			} else {
				return nil, fmt.Errorf("array type schema must have items defined")
			}

		case "object":
			// 对象类型的处理：递归处理对象的 properties，生成嵌套消息
			nestedMessage, err := convertObjectToProtoMessage(fieldName, schema)
			if err != nil {
				return nil, err
			}
			fmt.Println("nestedMessage:", nestedMessage.Name)
			return &protobuf.ProtoField{
				Name: fieldName,
				Type: nestedMessage.Name, // 引用嵌套的消息名
			}, nil

		default:
			// 如果遇到未知类型，返回错误
			return nil, fmt.Errorf("unsupported schema type: %s", mainType)
		}
	} else {
		return nil, fmt.Errorf("schema type is missing for field: %s", fieldName)
	}

	return &protobuf.ProtoField{
		Name: fieldName,
		Type: protoType,
	}, nil
}

// handleOneOf 处理 oneOf 或 anyOf 组合类型并生成 Protobuf 的 oneof 字段
func handleOneOf(fieldName string, schemas openapi3.SchemaRefs) (*protobuf.ProtoField, error) {
	var oneOfFields []protobuf.ProtoField

	for idx, schemaRef := range schemas {
		schema := schemaRef.Value
		if schema == nil {
			continue
		}

		// 为每个 oneOf 选项生成一个唯一字段名称
		fieldNameVariant := fmt.Sprintf("%s_option%d", fieldName, idx+1)

		// 将 schema 转换为 Protobuf 字段
		field, err := convertSchemaToProtoField(fieldNameVariant, schema)
		if err != nil {
			return nil, err
		}

		oneOfFields = append(oneOfFields, *field)
	}

	// 返回一个包含 oneof 字段的 Protobuf 结构
	return &protobuf.ProtoField{
		Name:  fieldName,
		OneOf: oneOfFields,
	}, nil
}

// handleAllOf 处理 allOf 组合类型并生成 Protobuf 的嵌套消息
func handleAllOf(fieldName string, schemas openapi3.SchemaRefs) (*protobuf.ProtoField, error) {
	message := &protobuf.ProtoMessage{Name: fieldName}

	for _, schemaRef := range schemas {
		schema := schemaRef.Value
		if schema == nil {
			continue
		}

		// 遍历 allOf 中的每个 schema，并将其字段合并到 message 中
		for propName, propSchemaRef := range schema.Properties {
			field, err := convertSchemaToProtoField(propName, propSchemaRef.Value)
			if err != nil {
				return nil, err
			}
			message.Fields = append(message.Fields, *field)
		}
	}

	return &protobuf.ProtoField{
		Name: message.Name,
		Type: message.Name, // 引用生成的 message
	}, nil
}

// convertComplexSchemaToProtoMessage 处理 OpenAPI 中的 oneOf/anyOf/allOf 类型
func convertComplexSchemaToProtoMessage(fieldName string, schema *openapi3.Schema) (*protobuf.ProtoMessage, error) {
	message := &protobuf.ProtoMessage{Name: fieldName}

	var complexFields []*protobuf.ProtoField
	for _, subSchemaRef := range append(schema.OneOf, schema.AnyOf...) {
		subSchema := subSchemaRef.Value
		if subSchema != nil {
			field, err := convertSchemaToProtoField(fieldName, subSchema)
			if err != nil {
				return nil, err
			}
			complexFields = append(complexFields, field)
		}
	}

	if len(complexFields) > 0 {
		for idx, field := range complexFields {
			field.Name = fmt.Sprintf("%s_option%d", fieldName, idx+1) // 创建唯一字段名
			message.Fields = append(message.Fields, *field)
		}
	}

	return message, nil
}

// convertObjectToProtoMessage 将 OpenAPI object 类型的 schema 转换为 Protobuf 嵌套消息
func convertObjectToProtoMessage(messageName string, schema *openapi3.Schema) (*protobuf.ProtoMessage, error) {
	// 创建一个新的 Protobuf 消息
	message := &protobuf.ProtoMessage{Name: messageName}

	// 遍历对象的 properties 并生成 Protobuf 字段
	for propName, propSchemaRef := range schema.Properties {
		// 检查是否存在 $ref
		if propSchemaRef.Ref != "" {
			// 处理 $ref，提取引用的消息名
			refMessageName := extractMessageNameFromRef(propSchemaRef.Ref)
			field := protobuf.ProtoField{
				Name: propName,
				Type: refMessageName, // 引用的 Protobuf 消息类型
			}
			message.Fields = append(message.Fields, field)
		} else if propSchemaRef.Value != nil {
			// 递归调用 convertSchemaToProtoField 来生成字段
			field, err := convertSchemaToProtoField(propName, propSchemaRef.Value)
			if err != nil {
				return nil, err
			}

			message.Fields = append(message.Fields, *field)
		}
	}

	// 返回生成的 Protobuf 嵌套消息
	return message, nil
}

// getPrimaryType 返回主类型，优先排除 "null"
func getPrimaryType(types *openapi3.Types) (string, error) {
	if types == nil || len(*types) == 0 {
		return "", fmt.Errorf("schema type is undefined")
	}

	// 优先选择非 "null" 类型
	for _, t := range *types {
		if t != "null" {
			return t, nil
		}
	}

	// 如果全是 null 类型，返回错误
	return "", fmt.Errorf("no valid non-null type found")
}

// convertSimpleTypeToProto 将简单的 OpenAPI 类型转换为 Protobuf 类型
func convertSimpleTypeToProto(openapiType string) string {
	switch openapiType {
	case "string":
		return "string"
	case "integer":
		return "int32" // 根据实际需求，可以使用 int32 或 int64
	case "boolean":
		return "bool"
	default:
		return "string" // 默认处理为字符串
	}
}

// getFirstType 获取 Types 中的第一个有效类型
func getFirstType(types *openapi3.Types) string {
	if types != nil && len(*types) > 0 {
		return (*types)[0]
	}
	return ""
}

// extractMessageNameFromRef 从 $ref 中提取消息名
func extractMessageNameFromRef(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1] // 返回最后一部分，通常是引用的名称
}

// getServiceName 从标签中获取服务名称
func getServiceName(tags []string) string {
	if len(tags) > 0 {
		return tags[0]
	}
	return "DefaultService"
}

// mergeProtoMessage 合并传入的 ProtoMessage 到现有的 ProtoFile.Messages 中
func (c *ProtoConverter) addMessageToProto(message *protobuf.ProtoMessage) error {
	// 查找是否已有同名的消息
	var existingMessage *protobuf.ProtoMessage
	for _, msg := range c.ProtoFile.Messages {
		if msg.Name == message.Name {
			existingMessage = msg
			break
		}
	}

	// 如果找到现有消息，进行字段去重合并
	if existingMessage != nil {
		// 去重合并字段
		fieldNames := make(map[string]struct{})
		for _, field := range existingMessage.Fields {
			fieldNames[field.Name] = struct{}{} // 记录已有字段名称
		}

		for _, newField := range message.Fields {
			if _, exists := fieldNames[newField.Name]; !exists {
				existingMessage.Fields = append(existingMessage.Fields, newField)
				fieldNames[newField.Name] = struct{}{} // 更新记录
			}
		}

	} else {
		// 如果没有同名的消息，直接添加到 ProtoFile.Messages 中
		c.ProtoFile.Messages = append(c.ProtoFile.Messages, message)
	}

	return nil
}
