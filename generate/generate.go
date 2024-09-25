package generate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/swagger-generate/swagger2idl/protobuf"
)

// ConvertToProtoFile 将 ProtoFile 结构转换为 Proto 文件内容
func ConvertToProtoFile(protoFile *protobuf.ProtoFile) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("syntax = \"proto3\";\n\n"))
	sb.WriteString(fmt.Sprintf("package %s;\n\n", protoFile.PackageName))

	// 对消息按名称排序
	sort.Slice(protoFile.Messages, func(i, j int) bool {
		return protoFile.Messages[i].Name < protoFile.Messages[j].Name
	})

	// 生成消息
	for _, message := range protoFile.Messages {
		sb.WriteString(fmt.Sprintf("message %s {\n", message.Name))

		// 对字段按名称排序
		sort.Slice(message.Fields, func(i, j int) bool {
			return message.Fields[i].Name < message.Fields[j].Name
		})

		for i, field := range message.Fields {
			repeated := ""
			if field.Repeated {
				repeated = "repeated "
			}
			sb.WriteString(fmt.Sprintf("  %s%s %s = %d;\n", repeated, field.Type, field.Name, i+1))
		}
		sb.WriteString("}\n\n")
	}

	// 对服务按名称排序
	sort.Slice(protoFile.Services, func(i, j int) bool {
		return protoFile.Services[i].Name < protoFile.Services[j].Name
	})

	// 生成服务
	for _, service := range protoFile.Services {
		sb.WriteString(fmt.Sprintf("service %s {\n", service.Name))

		// 对方法按名称排序
		sort.Slice(service.Methods, func(i, j int) bool {
			return service.Methods[i].Name < service.Methods[j].Name
		})

		for _, method := range service.Methods {
			sb.WriteString(fmt.Sprintf("  rpc %s(%s) returns (%s);\n", method.Name, method.Input, method.Output))
		}
		sb.WriteString("}\n\n")
	}

	return sb.String()
}
