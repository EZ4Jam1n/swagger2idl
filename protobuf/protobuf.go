package protobuf

// ProtoFile 代表一个完整的 Proto 文件
type ProtoFile struct {
	PackageName string          // Proto 文件的包名
	Messages    []*ProtoMessage // Proto 消息
	Services    []ProtoService  // Proto 服务
}

// ProtoMessage 代表一个 Proto 消息
type ProtoMessage struct {
	Name   string
	Fields []ProtoField
}

// ProtoField 代表一个 Proto 字段
type ProtoField struct {
	Name     string
	Type     string
	Repeated bool
	OneOf  []ProtoField  // oneof 字段集合，用于表示 oneOf 类型
	Fields []ProtoField  // 嵌套字段集合（例如对象类型）
	Options  []*FieldOption // 新增 options 字段
}

// FieldOption 代表 Proto 字段的选项
type FieldOption struct {
	Name  string
	Value interface{}
}

// ProtoMethod 代表一个 Proto 方法
type ProtoMethod struct {
	Name   string
	Input  string // 输入消息类型
	Output string // 输出消息类型
}

// ProtoService 代表一个 Proto 服务
type ProtoService struct {
	Name    string
	Methods []ProtoMethod
}
