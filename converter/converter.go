package converter

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/swagger-generate/swagger2idl/protobuf"
)

// ProtoConverter 结构体，用于转换 OpenAPI 规范为 Proto 文件
type ProtoConverter struct {
	ProtoFile       *protobuf.ProtoFile
	converterOption *ConvertOption
}

// ConvertOption 添加一个转换选项结构体
type ConvertOption struct {
	//是否生成openapi注释
	openapiOption bool

	apiOption bool
}

// NewProtoConverter 创建并初始化 ProtoConverter
func NewProtoConverter(packageName string) *ProtoConverter {
	return &ProtoConverter{
		ProtoFile: &protobuf.ProtoFile{
			PackageName: packageName,
			Messages:    []*protobuf.ProtoMessage{},
			Services:    []*protobuf.ProtoService{},
		},
		converterOption: &ConvertOption{
			openapiOption: false,
			apiOption:     true,
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

	if c.converterOption.openapiOption {
		c.AddProtoImport("openapi.proto")
	}
	if c.converterOption.apiOption {
		c.AddProtoImport("api.proto")
	}

	return nil
}

// convertComponentsToProtoMessages 转换 OpenAPI components 为 Proto 消息并存入 ProtoFile
func (c *ProtoConverter) convertComponentsToProtoMessages(components *openapi3.Components) error {
	for name, schemaRef := range components.Schemas {
		schema := schemaRef.Value
		message, err := c.ConvertSchemaToProtoMessage(schema, name)
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

// ConvertPathsToProtoServices 将 OpenAPI 的路径项转换为 Proto 服务
func (c *ProtoConverter) ConvertPathsToProtoServices(paths *openapi3.Paths) ([]*protobuf.ProtoService, error) {
	var services []*protobuf.ProtoService

	// 定义 HTTP 方法到 option 的映射
	methodToOption := map[string]string{
		"GET":    "api.get",
		"POST":   "api.post",
		"PUT":    "api.put",
		"PATCH":  "api.patch",
		"DELETE": "api.delete",
	}

	for path, pathItem := range paths.Map() {
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
				protoMethod := &protobuf.ProtoMethod{
					Name:   methodName,
					Input:  inputMessage,
					Output: outputMessage,
				}

				if c.converterOption.apiOption {

					// 根据 HTTP 方法获取 option
					if optionName, ok := methodToOption[method]; ok {
						option := &protobuf.Option{
							Name:  optionName,
							Value: fmt.Sprintf("%q", ConvertPath(path)),
						}
						protoMethod.Options = append(protoMethod.Options, option)
					}
				}

				if c.converterOption.openapiOption {
					optionStr := structToProtobuf(operation, "  ")

					// 添加 openapi.schema 注解
					schemaOption := &protobuf.Option{
						Name:  "openapi.operation",
						Value: optionStr,
					}
					protoMethod.Options = append(protoMethod.Options, schemaOption)

				}
				service.Methods = append(service.Methods, protoMethod)
			}
		}
	}

	return services, nil
}

// generateRequestMessage 生成请求消息
func (c *ProtoConverter) generateRequestMessage(operation *openapi3.Operation) (string, error) {
	messageName := operation.OperationID + "Request"
	message := &protobuf.ProtoMessage{Name: messageName}

	if operation.RequestBody == nil && len(operation.Parameters) == 0 {
		c.AddProtoImport("google.protobuf.Empty")
		return "google.protobuf.Empty", nil
	}

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
					requestMsg, err := c.ConvertSchemaToProtoMessage(schema.Value, messageName)
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
				var field *protobuf.ProtoField

				// 动态判断参数类型
				schemaType := getFirstType(param.Value.Schema.Value.Type)

				switch param.Value.In {
				case "path", "cookie", "header", "query":
					// 动态根据不同类型来生成对应的Proto类型
					switch schemaType {
					case "string":
						fieldType := "string"
						field = &protobuf.ProtoField{
							Name: param.Value.Name,
							Type: fieldType,
						}
					case "number", "integer":
						fieldType := "int32" // 或 "int64" 视情况而定
						field = &protobuf.ProtoField{
							Name: param.Value.Name,
							Type: fieldType,
						}
					case "boolean":
						fieldType := "bool"
						field = &protobuf.ProtoField{
							Name: param.Value.Name,
							Type: fieldType,
						}
					case "array":
						// 假设数组元素是简单类型，获取items的类型并处理
						arrayType := convertSimpleTypeToProto(getFirstType(param.Value.Schema.Value.Items.Value.Type))
						fieldType := "repeated " + arrayType
						field = &protobuf.ProtoField{
							Name: param.Value.Name,
							Type: fieldType,
						}
					case "object":
						// 如果是对象类型，需要递归生成子消息
						objectMessage, err := c.ConvertSchemaToProtoMessage(param.Value.Schema.Value, param.Value.Name)
						if err != nil {
							return "", err
						}
						c.addMessageToProto(objectMessage)

						field = &protobuf.ProtoField{
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
	if operation.Responses == nil {
		return "", nil
	}

	// 获取所有的 responseRef
	responses := operation.Responses.Map()
	responseCount := len(responses)

	// 如果只有一个 responseRef，直接处理并返回，不需要封装
	if responseCount == 1 {
		for statusCode, responseRef := range responses {
			return c.processSingleResponse(statusCode, responseRef, operation)
		}
	}

	// 如果有多个 responseRef，创建一个 wrapper message
	wrapperMessageName := operation.OperationID
	wrapperMessage := &protobuf.ProtoMessage{
		Name: wrapperMessageName,
	}

	// 遍历所有的 responseRef
	for statusCode, responseRef := range responses {
		// 处理每一个 responseRef
		if responseRef.Value != nil && responseRef.Value.Content != nil {
			messageName, err := c.processSingleResponse(statusCode, responseRef, operation)
			if err != nil {
				return "", err
			}
			// 添加每个 response 到 wrapper message 中
			field := &protobuf.ProtoField{
				Name: "response_" + statusCode,
				Type: messageName,
			}
			wrapperMessage.Fields = append(wrapperMessage.Fields, field)
		}
	}

	// 将封装好的 wrapperMessage 添加到 proto 中
	c.addMessageToProto(wrapperMessage)

	// 返回封装好的消息名称
	return wrapperMessageName, nil
}

// processSingleResponse 处理单个 responseRef 并生成对应的 Protobuf message
func (c *ProtoConverter) processSingleResponse(statusCode string, responseRef *openapi3.ResponseRef, operation *openapi3.Operation) (string, error) {
	// 如果是 $ref，直接返回引用的消息名
	if responseRef.Ref != "" {
		return extractMessageNameFromRef(responseRef.Ref), nil
	}

	if responseRef.Value == nil || responseRef.Value.Content == nil {
		c.AddProtoImport("google.protobuf.Empty")
		return "google.protobuf.Empty", nil
	}

	// 获取 response 的值
	response := responseRef.Value
	for _, mediaType := range response.Content {
		schema := mediaType.Schema
		if schema != nil && schema.Value != nil {
			// 如果 schema 有 $ref，生成引用的消息名并返回
			if schema.Ref != "" {
				return extractMessageNameFromRef(schema.Ref), nil
			}

			// 生成对应的 Protobuf message 名称
			messageName := operation.OperationID + "Response_" + statusCode

			// 根据 schema 生成 Proto message
			responseMsg, err := c.ConvertSchemaToProtoMessage(schema.Value, messageName)
			if err != nil {
				return "", err
			}

			// 将生成的 response message 添加到 proto 中
			c.addMessageToProto(responseMsg)

			// 返回生成的消息名称
			return responseMsg.Name, nil
		}
	}

	return "", nil
}

// findOrCreateService 查找服务，如果不存在则创建新服务
func findOrCreateService(services *[]*protobuf.ProtoService, serviceName string) *protobuf.ProtoService {
	for i := range *services {
		if (*services)[i].Name == serviceName {
			return (*services)[i]
		}
	}

	// 如果未找到，创建新服务
	newService := &protobuf.ProtoService{Name: serviceName}
	*services = append(*services, newService) // 修改指向原始切片的指针
	return (*services)[len(*services)-1]
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

// 处理嵌套的 Map 类型
func (c *ProtoConverter) handleNestedAdditionalProperties(schema *openapi3.Schema, defaultName string) (*protobuf.ProtoField, error) {
	if schema.AdditionalProperties.Schema != nil {
		keyType := "string" // Map 的键通常是字符串
		var valueType string

		// 如果 additionalProperties 是 object，递归生成 message
		if schema.AdditionalProperties.Schema.Value.Type.Includes("object") {
			// 生成嵌套消息名
			nestedMessageName := defaultName + "Nested"

			// 递归生成嵌套的 Protobuf 消息
			nestedMsg, err := c.ConvertSchemaToProtoMessage(schema.AdditionalProperties.Schema.Value, nestedMessageName)
			if err != nil {
				return &protobuf.ProtoField{}, err
			}

			// 将嵌套消息添加到 ProtoFile 中
			c.addMessageToProto(nestedMsg)

			// 使用嵌套消息的名称作为 Map 的值类型
			valueType = nestedMsg.Name
		} else {
			// 处理简单类型
			valueType = convertSimpleTypeToProto(getFirstType(schema.AdditionalProperties.Schema.Value.Type))
		}

		// 使用默认名称作为字段名
		fieldName := schema.Title
		if fieldName == "" {
			fieldName = defaultName
		}

		// 返回 map 类型的字段
		field := &protobuf.ProtoField{
			Name: fieldName,
			Type: fmt.Sprintf("map<%s, %s>", keyType, valueType),
		}
		return field, nil
	}
	return &protobuf.ProtoField{}, fmt.Errorf("additionalProperties must have a schema")
}

// generateMethodName 生成方法名，基于 OperationID 和 HTTP 方法
func generateMethodName(operationID, method string) string {
	if operationID != "" {
		return operationID
	}
	// 如果没有 OperationID，用 HTTP 方法生成
	return strings.Title(strings.ToLower(method)) + "Method"
}

// ConvertSchemaToProtoMessage 将 OpenAPI Schema 转换为 Protobuf 消息
func (c *ProtoConverter) ConvertSchemaToProtoMessage(schema *openapi3.Schema, name string) (*protobuf.ProtoMessage, error) {
	protoMsg := &protobuf.ProtoMessage{Name: name}

	if c.converterOption.openapiOption {
		optionStr := structToProtobuf(schema, "  ")

		// 添加 openapi.schema 注解
		schemaOption := &protobuf.Option{
			Name:  "openapi.schema",
			Value: optionStr,
		}
		protoMsg.Options = append(protoMsg.Options, schemaOption)
	}

	// 检查是否是对象类型
	if schema.Type != nil && schema.Type.Includes("object") {
		// 遍历对象的属性，将每个属性转换为 Protobuf 字段
		for propName, propSchema := range schema.Properties {
			if propSchema.Ref != "" {
				propName = extractMessageNameFromRef(propSchema.Ref)
			}
			field, err := c.convertSchemaToProtoField(propName, propSchema.Value)
			if err != nil {
				return nil, err
			}
			protoMsg.Fields = append(protoMsg.Fields, field)
		}

		// 处理 additionalProperties（如果定义了）
		if schema.AdditionalProperties.Schema != nil {
			nestedField, err := c.handleNestedAdditionalProperties(schema, name+"Nested")
			if err != nil {
				return nil, err
			}
			protoMsg.Fields = append(protoMsg.Fields, nestedField)
		}
	} else {
		// 处理非对象类型
		fieldType, err := convertSchemaToProtoType(schema)
		if err != nil {
			return nil, err
		}
		field := &protobuf.ProtoField{Name: "value", Type: fieldType}
		protoMsg.Fields = append(protoMsg.Fields, field)
	}

	return protoMsg, nil
}

func structToProtobuf(value interface{}, indent string) string {
	var sb strings.Builder
	v := reflect.ValueOf(value)
	t := reflect.TypeOf(value)

	// 如果是指针，获取实际值
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "" // 跳过 nil 指针
		}
		v = v.Elem()
		t = t.Elem()
	}

	// 处理切片类型
	if v.Kind() == reflect.Slice {
		if v.Len() == 0 {
			return "" // 跳过空切片
		}
		sb.WriteString("[\n")
		for i := 0; i < v.Len(); i++ {
			sb.WriteString(fmt.Sprintf("%s  ", indent))
			sb.WriteString(structToProtobuf(v.Index(i).Interface(), indent+"  "))
			if i < v.Len()-1 {
				sb.WriteString(",\n")
			}
		}
		sb.WriteString(fmt.Sprintf("\n%s]", indent))
		return sb.String()
	}

	// 处理映射类型
	if v.Kind() == reflect.Map {
		if v.Len() == 0 {
			return "" // 跳过空 map
		}
		sb.WriteString("{\n")
		for _, key := range v.MapKeys() {
			sb.WriteString(fmt.Sprintf("%s  %v: ", indent, key))
			sb.WriteString(structToProtobuf(v.MapIndex(key).Interface(), indent+"  "))
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf("%s}", indent))
		return sb.String()
	}

	// 处理结构体类型
	if v.Kind() == reflect.Struct {
		sb.WriteString("{\n")
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := t.Field(i)

			// 跳过未导出的字段
			if !field.CanInterface() {
				continue
			}

			// 跳过零值字段
			if isZeroValue(field) {
				continue
			}

			fieldName := fieldType.Tag.Get("json")
			if fieldName == "" {
				fieldName = fieldType.Name // 如果没有 json 标签，使用字段名
			}
			fieldName = strings.Split(fieldName, ",")[0] // 去掉 json 标签中的选项，例如 "omitempty"

			// 使用字段名作为 Protobuf 的键
			sb.WriteString(fmt.Sprintf("%s  %s: ", indent, fieldName))

			// 递归处理字段
			sb.WriteString(structToProtobuf(field.Interface(), indent+"  "))
			sb.WriteString(";\n")
		}
		sb.WriteString(fmt.Sprintf("%s}", indent))
		return sb.String()
	}

	// 处理其他基本类型
	switch v.Kind() {
	case reflect.String:
		if v.String() == "" {
			return "" // 跳过空字符串
		}
		return fmt.Sprintf("\"%s\"", v.String())
	case reflect.Int, reflect.Int64, reflect.Int32:
		if v.Int() == 0 {
			return "" // 跳过 0
		}
		return fmt.Sprintf("%d", v.Int())
	case reflect.Float64:
		if v.Float() == 0 {
			return "" // 跳过 0.0
		}
		return fmt.Sprintf("%f", v.Float())
	case reflect.Bool:
		if !v.Bool() {
			return "" // 跳过 false
		}
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Ptr:
		if !v.IsNil() {
			return structToProtobuf(v.Interface(), indent)
		}
		return ""
	default:
		// 跳过零值
		if !v.IsValid() || v.IsZero() {
			return ""
		}
		return fmt.Sprintf("\"%v\"", v.Interface())
	}
}

// isZeroValue 判断一个值是否为零值
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
		// 递归判断结构体是否所有字段都是零值
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

// convertSchemaToProtoField 根据 OpenAPI schema 生成 Protobuf 字段，递归处理嵌套类型
func (c *ProtoConverter) convertSchemaToProtoField(fieldName string, schema *openapi3.Schema) (*protobuf.ProtoField, error) {
	var protoType string

	// 检查 schema 是否有组合类型定义 (oneOf, anyOf, allOf)
	if len(schema.OneOf) > 0 {
		return c.handleOneOf(fieldName, schema.OneOf)
	}
	if len(schema.AnyOf) > 0 {
		return c.handleOneOf(fieldName, schema.AnyOf) // 与 oneOf 逻辑相似
	}
	if len(schema.AllOf) > 0 {
		return c.handleAllOf(fieldName, schema.AllOf)
	}

	// 处理对象类型的 additionalProperties，将其转换为 Protobuf map
	if schema.Type != nil && schema.Type.Includes("object") {
		if schema.AdditionalProperties.Schema != nil {
			// 处理 additionalProperties 为 map 的情况
			keyType := "string" // Protobuf 的 map 键类型通常是 string
			valueSchema := schema.AdditionalProperties.Schema.Value
			valueType, err := convertSchemaToProtoType(valueSchema)
			if err != nil {
				return nil, err
			}

			protoType = fmt.Sprintf("map<%s, %s>", keyType, valueType)
		} else {
			// 否则处理为嵌套的 message
			nestedMessage, err := c.convertObjectToProtoMessage(fieldName, schema)
			if err != nil {
				return nil, err
			}
			return &protobuf.ProtoField{
				Name: fieldName,
				Type: nestedMessage.Name, // 引用嵌套的消息名
			}, nil
		}
	} else {
		// 处理其他基础类型
		fieldType, err := convertSchemaToProtoType(schema)
		if err != nil {
			return nil, err
		}
		protoType = fieldType
	}

	field := &protobuf.ProtoField{
		Name: fieldName,
		Type: protoType,
	}

	if c.converterOption.openapiOption {
		optionStr := structToProtobuf(schema, "    ")

		// 添加 openapi.schema 注解
		schemaOption := &protobuf.Option{
			Name:  "openapi.property",
			Value: optionStr,
		}
		field.Options = append(field.Options, schemaOption)
	}
	return field, nil
}

// convertSchemaToProtoType 将 OpenAPI 基本类型转换为 Protobuf 类型
func convertSchemaToProtoType(schema *openapi3.Schema) (string, error) {
	if len(schema.Type.Slice()) == 0 {
		return "", fmt.Errorf("schema type is missing")
	}

	mainType, err := getPrimaryType(schema.Type)
	if err != nil {
		return "", err
	}

	switch mainType {
	case "string":
		return "string", nil
	case "integer":
		if schema.Format == "int64" {
			return "int64", nil
		}
		return "int32", nil
	case "boolean":
		return "bool", nil
	case "number":
		if schema.Format == "float" {
			return "float", nil
		} else if schema.Format == "double" {
			return "double", nil
		}
		return "", fmt.Errorf("unsupported number format: %s", schema.Format)
	case "array":
		if schema.Items != nil && schema.Items.Value != nil {
			itemType := convertSimpleTypeToProto(getFirstType(schema.Items.Value.Type))
			return "repeated " + itemType, nil
		}
		return "", fmt.Errorf("array type schema must have items defined")
	default:
		return "", fmt.Errorf("unsupported schema type: %s", mainType)
	}
}

// handleOneOf 处理 oneOf 或 anyOf 组合类型并生成 Protobuf 的 oneof 字段
func (c *ProtoConverter) handleOneOf(fieldName string, schemas openapi3.SchemaRefs) (*protobuf.ProtoField, error) {
	var oneOfFields []*protobuf.ProtoField

	for idx, schemaRef := range schemas {
		schema := schemaRef.Value
		if schema == nil {
			continue
		}

		// 为每个 oneOf 选项生成一个唯一字段名称
		fieldNameVariant := fmt.Sprintf("%s_option%d", fieldName, idx+1)

		// 将 schema 转换为 Protobuf 字段
		field, err := c.convertSchemaToProtoField(fieldNameVariant, schema)
		if err != nil {
			return nil, err
		}

		oneOfFields = append(oneOfFields, field)
	}

	// 返回一个包含 oneof 字段的 Protobuf 结构
	return &protobuf.ProtoField{
		Name:  fieldName,
		OneOf: oneOfFields,
	}, nil
}

// handleAllOf 处理 allOf 组合类型并生成 Protobuf 的嵌套消息
func (c *ProtoConverter) handleAllOf(fieldName string, schemas openapi3.SchemaRefs) (*protobuf.ProtoField, error) {
	message := &protobuf.ProtoMessage{Name: fieldName}

	for _, schemaRef := range schemas {
		schema := schemaRef.Value
		if schema == nil {
			continue
		}

		// 遍历 allOf 中的每个 schema，并将其字段合并到 message 中
		for propName, propSchemaRef := range schema.Properties {
			field, err := c.convertSchemaToProtoField(propName, propSchemaRef.Value)
			if err != nil {
				return nil, err
			}
			message.Fields = append(message.Fields, field)
		}
	}

	return &protobuf.ProtoField{
		Name: message.Name,
		Type: message.Name, // 引用生成的 message
	}, nil
}

// convertComplexSchemaToProtoMessage 处理 OpenAPI 中的 oneOf/anyOf/allOf 类型
func (c *ProtoConverter) convertComplexSchemaToProtoMessage(fieldName string, schema *openapi3.Schema) (*protobuf.ProtoMessage, error) {
	message := &protobuf.ProtoMessage{Name: fieldName}

	var complexFields []*protobuf.ProtoField
	for _, subSchemaRef := range append(schema.OneOf, schema.AnyOf...) {
		subSchema := subSchemaRef.Value
		if subSchema != nil {
			field, err := c.convertSchemaToProtoField(fieldName, subSchema)
			if err != nil {
				return nil, err
			}
			complexFields = append(complexFields, field)
		}
	}

	if len(complexFields) > 0 {
		for idx, field := range complexFields {
			field.Name = fmt.Sprintf("%s_option%d", fieldName, idx+1) // 创建唯一字段名
			message.Fields = append(message.Fields, field)
		}
	}

	return message, nil
}

// convertObjectToProtoMessage 将 OpenAPI object 类型的 schema 转换为 Protobuf 嵌套消息
func (c *ProtoConverter) convertObjectToProtoMessage(messageName string, schema *openapi3.Schema) (*protobuf.ProtoMessage, error) {
	// 创建一个新的 Protobuf 消息
	message := &protobuf.ProtoMessage{Name: messageName}

	// 遍历对象的 properties 并生成 Protobuf 字段
	for propName, propSchemaRef := range schema.Properties {
		// 检查是否存在 $ref
		if propSchemaRef.Ref != "" {
			// 处理 $ref，提取引用的消息名
			refMessageName := extractMessageNameFromRef(propSchemaRef.Ref)
			field := &protobuf.ProtoField{
				Name: propName,
				Type: refMessageName, // 引用的 Protobuf 消息类型
			}
			message.Fields = append(message.Fields, field)
		} else if propSchemaRef.Value != nil {
			// 递归调用 convertSchemaToProtoField 来生成字段
			field, err := c.convertSchemaToProtoField(propName, propSchemaRef.Value)
			if err != nil {
				return nil, err
			}

			message.Fields = append(message.Fields, field)
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

// AddProtoImport 添加一个新的导入文件到 ProtoConverter 的 ProtoFile
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

// ConvertPath converts a path with {} to use : for parameters
func ConvertPath(path string) string {
	// 正则表达式匹配 {} 中的内容
	re := regexp.MustCompile(`\{(\w+)\}`)
	// 使用 : 替换 {param} 为 :param
	result := re.ReplaceAllString(path, ":$1")
	return result
}
