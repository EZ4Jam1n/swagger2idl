package generate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/swagger-generate/swagger2idl/protobuf"
)

// Encoder 用于处理编码的上下文
type Encoder struct {
	dst *strings.Builder // 输出的目标
}

// ConvertToProtoFile 将 ProtoFile 结构转换为 Proto 文件内容
func ConvertToProtoFile(protoFile *protobuf.ProtoFile) string {
	var sb strings.Builder
	encoder := &Encoder{dst: &sb}

	encoder.dst.WriteString(fmt.Sprintf("syntax = \"proto3\";\n\n"))
	encoder.dst.WriteString(fmt.Sprintf("package %s;\n\n", protoFile.PackageName))

	// 生成 imports
	for _, importFile := range protoFile.Imports {
		encoder.dst.WriteString(fmt.Sprintf("import \"%s\";\n", importFile))
	}
	if len(protoFile.Imports) > 0 {
		encoder.dst.WriteString("\n")
	}

	// 生成文件级别的选项
	for key, value := range protoFile.Options {
		encoder.dst.WriteString(fmt.Sprintf("option %s = %s;\n", key, Stringify(value)))
	}
	if len(protoFile.Options) > 0 {
		encoder.dst.WriteString("\n")
	}

	// 对消息按名称排序
	sort.Slice(protoFile.Messages, func(i, j int) bool {
		return protoFile.Messages[i].Name < protoFile.Messages[j].Name
	})

	// 生成消息
	for _, message := range protoFile.Messages {
		encoder.dst.WriteString(fmt.Sprintf("message %s {\n", message.Name))

		if len(message.Options) > 0 {
			encoder.dst.WriteString("  option")
		}
		// 生成消息级别选项
		for _, option := range message.Options {
			encoder.dst.WriteString(option.Value.(string))
			encoder.encodeFieldOption(option)
			encoder.dst.WriteString(";\n")
		}

		// 对字段按名称排序
		sort.Slice(message.Fields, func(i, j int) bool {
			return message.Fields[i].Name < message.Fields[j].Name
		})

		for i, field := range message.Fields {
			repeated := ""
			if field.Repeated {
				repeated = "repeated "
			}
			encoder.dst.WriteString(fmt.Sprintf("  %s%s %s = %d", repeated, field.Type, field.Name, i+1))

			// 生成字段级别的选项
			if len(field.Options) > 0 {
				encoder.dst.WriteString(" [\n    ")
				for j, option := range field.Options {
					encoder.encodeFieldOption(option)
					if j < len(field.Options)-1 {
						encoder.dst.WriteString(", ")
					}
				}
				encoder.dst.WriteString("]")
			}

			encoder.dst.WriteString(";\n")
		}
		encoder.dst.WriteString("}\n\n")
	}

	// 对服务按名称排序
	sort.Slice(protoFile.Services, func(i, j int) bool {
		return protoFile.Services[i].Name < protoFile.Services[j].Name
	})

	// 生成服务
	for _, service := range protoFile.Services {
		encoder.dst.WriteString(fmt.Sprintf("service %s {\n", service.Name))

		// 对方法按名称排序
		sort.Slice(service.Methods, func(i, j int) bool {
			return service.Methods[i].Name < service.Methods[j].Name
		})

		for _, method := range service.Methods {
			encoder.dst.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s)", method.Name, method.Input, method.Output))
			if len(method.Options) > 0 {
				encoder.dst.WriteString(" {\n")
				for _, option := range method.Options {
					encoder.dst.WriteString("     option ")
					encoder.encodeFieldOption(option)
					encoder.dst.WriteString(";\n")
				}
				encoder.dst.WriteString("  }\n")
			} else {
				encoder.dst.WriteString(";\n")
			}
		}
		encoder.dst.WriteString("}\n\n")
	}

	return encoder.dst.String()
}

// encodeFieldOption 编码单个字段的 option
func (e *Encoder) encodeFieldOption(opt *protobuf.Option) error {
	// 输出 option 的名称
	fmt.Fprintf(e.dst, "(%s) = ", opt.Name) // 增加缩进，保持一致

	// 判断 option 的值是否是一个复杂结构
	switch value := opt.Value.(type) {
	case map[string]interface{}:
		// 如果是 map 类型，表示需要嵌套输出
		fmt.Fprintf(e.dst, "{\n")        // 在 { 后换行
		e.encodeFieldOptionMap(value, 6) // 输出 map 内容，传递当前缩进层次
		fmt.Fprintf(e.dst, "    }")      // 缩进并输出结束的 }，根据缩进层次决定缩进量
	default:
		fmt.Fprintf(e.dst, "%s", value) // 对于简单类型，直接输出
	}

	return nil
}

// encodeFieldOptionMap 编码复杂的 map 类型的 option 值
func (e *Encoder) encodeFieldOptionMap(optionMap map[string]interface{}, indent int) error {
	keys := make([]string, 0, len(optionMap))
	for k := range optionMap {
		keys = append(keys, k)
	}
	sort.Strings(keys) // 对键进行排序，确保输出顺序一致

	indentSpace := strings.Repeat(" ", indent) // 动态生成缩进空格

	for _, key := range keys {
		value := optionMap[key]
		// 输出键值对，适当缩进
		fmt.Fprintf(e.dst, "%s%s: %s", indentSpace, key, Stringify(value)) // 增加更深的缩进
		// 不在最后一项后面加分号，保持正确格式
		fmt.Fprintf(e.dst, ";\n")
	}

	return nil
}

func Stringify(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v) // 对字符串加引号
	case int, int64, float64:
		return fmt.Sprintf("%v", v) // 对数字直接输出
	case *uint64:
		return fmt.Sprintf("%d", *v) // 处理 *uint64 类型指针
	case []string:
		return fmt.Sprintf("[%s]", strings.Join(v, ", ")) // 对字符串数组输出为列表形式
	case []interface{}:
		// 处理任意类型的数组
		var strValues []string
		for _, item := range v {
			strValues = append(strValues, Stringify(item))
		}
		return fmt.Sprintf("[%s]", strings.Join(strValues, ", "))
	default:
		return fmt.Sprintf("%v", v) // 其他类型直接转为字符串
	}
}
