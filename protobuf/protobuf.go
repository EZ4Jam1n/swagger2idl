package protobuf

// ProtoFile 代表一个完整的 Proto 文件
type ProtoFile struct {
	PackageName string            // Proto 文件的包名
	Messages    []*ProtoMessage   // Proto 消息
	Services    []*ProtoService   // Proto 服务
	Enums       []*ProtoEnum      // Proto 枚举
	Imports     []string          // 引用的其他 Proto 文件
	Options     map[string]string // 文件级别的选项
}

// ProtoMessage 代表一个 Proto 消息
type ProtoMessage struct {
	Name    string
	Fields  []*ProtoField
	Options []*Option
}

// ProtoField 代表一个 Proto 字段
type ProtoField struct {
	Name     string
	Type     string
	Repeated bool
	OneOf    []*ProtoField // oneof 字段集合，用于表示 oneOf 类型
	Fields   []*ProtoField // 嵌套字段集合（例如对象类型）
	Options  []*Option     // 新增 options 字段
}

// Option 代表 Proto 字段的选项
type Option struct {
	Name  string
	Value interface{}
}

// ProtoMethod 代表一个 Proto 方法
type ProtoMethod struct {
	Name    string
	Input   string // 输入消息类型
	Output  string // 输出消息类型
	Options []*Option
}

// ProtoService 代表一个 Proto 服务
type ProtoService struct {
	Name    string
	Methods []*ProtoMethod
	Options []*Option
}

// ProtoEnum 代表一个 Proto 枚举
type ProtoEnum struct {
	Name    string            // 枚举名称
	Values  []*ProtoEnumValue // 枚举值
	Options []*Option
}

// ProtoEnumValue 代表一个 Proto 枚举值
type ProtoEnumValue struct {
	Name  string // 枚举值名称
	Value int32  // 枚举值对应的整数值
}
