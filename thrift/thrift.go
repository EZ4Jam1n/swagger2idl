package thrift

// ThriftFile represents a complete Thrift file
type ThriftFile struct {
	Namespace map[string]string // Namespace for the Thrift file
	Includes  []string          // List of included Thrift files
	Structs   []*ThriftStruct   // List of Thrift structs
	Unions    []*ThriftUnion    // List of Thrift unions
	Enums     []*ThriftEnum     // List of Thrift enums
	Constants []*ThriftConstant // List of constants
	Services  []*ThriftService  // List of Thrift services
}

// ThriftStruct represents a Thrift struct
type ThriftStruct struct {
	Name        string         // Name of the struct
	Description string         // Description of the struct
	Fields      []*ThriftField // List of fields in the struct
	Options     []*Option      // Options specific to this struct
}

// ThriftField represents a field in a Thrift struct or union
type ThriftField struct {
	ID          int       // Field ID for Thrift
	Name        string    // Name of the field
	Description string    // Description of the field
	Type        string    // Type of the field (Thrift types)
	Optional    bool      // Indicates if the field is optional
	Repeated    bool      // Indicates if the field is repeated (list)
	Options     []*Option // Additional options for this field
}

// ThriftUnion represents a Thrift union (similar to a struct but only one field can be set at a time)
type ThriftUnion struct {
	Name    string         // Name of the union
	Fields  []*ThriftField // List of fields in the union
	Options []*Option      // Options specific to this union
}

// ThriftEnum represents a Thrift enum
type ThriftEnum struct {
	Name        string             // Name of the enum
	Description string             // Description of the enum
	Values      []*ThriftEnumValue // Values within the enum
	Options     []*Option          // Enum-level options
}

// ThriftEnumValue represents a value in a Thrift enum
type ThriftEnumValue struct {
	Index int // Index of the enum value
	Value any // Enum values are integers in Thrift
}

// ThriftConstant represents a constant in Thrift
type ThriftConstant struct {
	Name  string      // Name of the constant
	Type  string      // Type of the constant
	Value interface{} // Value of the constant
}

// ThriftMethod represents a method in a Thrift service
type ThriftMethod struct {
	Name        string    // Name of the method
	Description string    // Description of the method
	Input       []string  // List of input fields for the method
	Output      string    // Output field for the method
	Options     []*Option // Options for the method
}

// ThriftService represents a Thrift service
type ThriftService struct {
	Name        string          // Name of the service
	Description string          // Description of the service
	Methods     []*ThriftMethod // List of methods in the service
	Options     []*Option       // Service-level options
}

// Option represents an option in a Thrift field or struct
type Option struct {
	Name  string      // Name of the option
	Value interface{} // Value of the option
}
