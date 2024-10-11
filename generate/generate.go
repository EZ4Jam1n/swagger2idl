package generate

// Generator is an interface for generating files
type Generator interface {
	Generate(fileContent interface{}) (string, error)
}
