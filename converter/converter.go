package converter

type Converter interface {
	Convert() error
	GetIdl() interface{}
}

// ConvertOption adds a struct for conversion options
type ConvertOption struct {
	OpenapiOption bool
	ApiOption     bool
	NamingOption  bool
}

var (
	MethodToOption = map[string]string{
		"GET":     "api.get",
		"POST":    "api.post",
		"PUT":     "api.put",
		"PATCH":   "api.patch",
		"DELETE":  "api.delete",
		"HEAD":    "api.head",
		"OPTIONS": "api.options",
	}
)
