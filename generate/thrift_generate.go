package generate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hertz-contrib/swagger-generate/swagger2idl/thrift"
)

// ThriftGenerate 用于处理 Thrift 文件的编码上下文
type ThriftGenerate struct {
	dst *strings.Builder // 输出目标
}

// NewThriftGenerate 创建一个新的 ThriftGenerate 实例
func NewThriftGenerate() *ThriftGenerate {
	return &ThriftGenerate{dst: &strings.Builder{}}
}

// Generate 将 ThriftFile 结构转换为 Thrift 文件内容
func (e *ThriftGenerate) Generate(fileContent interface{}) (string, error) {
	thriftFile, ok := fileContent.(*thrift.ThriftFile)
	if !ok {
		return "", fmt.Errorf("invalid type: expected *ThriftFile")
	}

	// 写入 namespace
	for language, ns := range thriftFile.Namespace {
		e.dst.WriteString(fmt.Sprintf("namespace %s %s\n", language, ns))
	}
	e.dst.WriteString("\n")

	// 生成 includes
	for _, include := range thriftFile.Includes {
		e.dst.WriteString(fmt.Sprintf("include \"%s\"\n", include))
	}
	if len(thriftFile.Includes) > 0 {
		e.dst.WriteString("\n")
	}

	// 生成 enums
	for _, enum := range thriftFile.Enums {
		e.encodeEnum(enum, 0)
	}

	// 生成 constants
	for _, constant := range thriftFile.Constants {
		e.encodeConstant(constant)
	}

	// 生成 structs
	for _, message := range thriftFile.Structs {
		e.encodeMessage(message, 0)
	}

	// 生成 unions
	for _, union := range thriftFile.Unions {
		e.encodeUnion(union, 0)
	}

	// 生成 services
	for _, service := range thriftFile.Services {
		e.encodeService(service)
	}

	return e.dst.String(), nil
}

// encodeEnum 编码枚举类型
func (e *ThriftGenerate) encodeEnum(enum *thrift.ThriftEnum, indentLevel int) {
	indent := strings.Repeat("    ", indentLevel)
	e.dst.WriteString(fmt.Sprintf("%senum %s {\n", indent, enum.Name))
	for _, value := range enum.Values {
		// Convert the value to a string
		valueStr := fmt.Sprintf("%v", value.Value)

		// Check if the value is a number and generate a name if necessary
		enumValueName := valueStr
		if _, err := strconv.Atoi(valueStr); err == nil {
			enumValueName = fmt.Sprintf("%s%s", enum.Name, valueStr)
		}

		e.dst.WriteString(fmt.Sprintf("%s  %s = %d;\n", indent, enumValueName, value.Index))
	}
	e.dst.WriteString(fmt.Sprintf("%s}\n\n", indent))
}

// encodeField 编码 struct 中的单个字段
func (e *ThriftGenerate) encodeField(field *thrift.ThriftField, index int, indentLevel int) {
	indent := strings.Repeat("    ", indentLevel)

	// 字段编号和类型
	fieldType := field.Type
	if field.Repeated {
		fieldType = fmt.Sprintf("list<%s>", field.Type)
	}

	// 处理可选字段
	optionalFlag := ""
	if field.Optional {
		optionalFlag = "optional "
	}

	// 使用提供的 index 赋值给字段
	e.dst.WriteString(fmt.Sprintf("%s%d: %s%s %s", indent, index, optionalFlag, fieldType, field.Name))

	// 字段选项
	if len(field.Options) > 0 {
		e.dst.WriteString(" (\n")
		for i, option := range field.Options {
			if i > 0 {
				e.dst.WriteString(",\n")
			}
			e.dst.WriteString(indent + "    ")
			e.encodeOption(option)
		}
		e.dst.WriteString("\n" + indent + ")\n")
	} else {
		e.dst.WriteString("\n")
	}
}

// encodeMessage 递归编码 structs，包括嵌套的 structs 和 enums
func (e *ThriftGenerate) encodeMessage(message *thrift.ThriftStruct, indentLevel int) {
	indent := strings.Repeat("    ", indentLevel)
	e.dst.WriteString(fmt.Sprintf("%sstruct %s {\n", indent, message.Name))

	// 字段：遍历字段并分配索引
	for i, field := range message.Fields {
		e.encodeField(field, i+1, indentLevel+1) // `i+1` 分配基于1的索引
	}

	e.dst.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// struct 选项
	if len(message.Options) > 0 {
		e.dst.WriteString(indent + "(")
		for i, option := range message.Options {
			if i > 0 {
				e.dst.WriteString(", ")
			}
			e.encodeOption(option)
		}
		e.dst.WriteString(")\n")
	}
}

// encodeService 编码服务定义
func (e *ThriftGenerate) encodeService(service *thrift.ThriftService) {
	// 服务注释
	e.dst.WriteString(fmt.Sprintf("service %s {\n", service.Name))

	// 方法
	for _, method := range service.Methods {
		e.encodeMethod(method)
	}

	e.dst.WriteString("}\n\n")

	// 服务选项
	if len(service.Options) > 0 {
		e.dst.WriteString("(")
		for i, option := range service.Options {
			if i > 0 {
				e.dst.WriteString(", ")
			}
			e.encodeOption(option)
		}
		e.dst.WriteString(")\n")
	}
}

// encodeUnion 编码 Thrift union
func (e *ThriftGenerate) encodeUnion(union *thrift.ThriftUnion, indentLevel int) {
	indent := strings.Repeat("    ", indentLevel)
	e.dst.WriteString(fmt.Sprintf("%sunion %s {\n", indent, union.Name))

	// 遍历 union 的字段
	for i, field := range union.Fields {
		e.encodeField(field, i+1, indentLevel+1) // `i+1` 用于分配1-based索引
	}

	e.dst.WriteString(fmt.Sprintf("%s}\n\n", indent))
}

// encodeMethod 编码服务中的方法
func (e *ThriftGenerate) encodeMethod(method *thrift.ThriftMethod) {
	// 方法签名
	e.dst.WriteString(fmt.Sprintf("    %s %s (", method.Output, method.Name))

	// 输入参数
	for i, input := range method.Input {
		if i > 0 {
			e.dst.WriteString(", ")
		}
		e.dst.WriteString(fmt.Sprintf("%d: %s req", i+1, input))
	}

	e.dst.WriteString(")")

	// 方法选项
	if len(method.Options) > 0 {
		e.dst.WriteString(" (\n")
		for i, option := range method.Options {
			if i > 0 {
				e.dst.WriteString(",\n")
			}
			e.dst.WriteString("        ")
			e.encodeOption(option)
		}
		e.dst.WriteString("\n    )")
	}

	e.dst.WriteString("\n")
}

// encodeConstant 编码 Thrift 常量
func (e *ThriftGenerate) encodeConstant(constant *thrift.ThriftConstant) {
	e.dst.WriteString(fmt.Sprintf("const %s %s = %v;\n", constant.Type, constant.Name, constant.Value))
}

// encodeOption 处理方法、struct 和字段选项的编码
func (e *ThriftGenerate) encodeOption(option *thrift.Option) {
	// 如果选项键以 "api" 开头，值使用双引号
	if strings.HasPrefix(option.Name, "api.") {
		e.dst.WriteString(fmt.Sprintf("%s = %s", option.Name, option.Value))
	} else if strings.HasPrefix(option.Name, "openapi.") {
		e.dst.WriteString(fmt.Sprintf("%s = '%s'", option.Name, option.Value))
	} else {
		e.dst.WriteString(fmt.Sprintf("%s = %s", option.Name, option.Value))
	}
}
