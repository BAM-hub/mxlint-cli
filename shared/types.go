package shared

type MxDocument struct {
	Name       string                 `yaml:"Name"`
	Type       string                 `yaml:"Type"`
	Path       string                 `yaml:"Path"`
	Hash       string                 `yaml:"Hash"`
	Id         string                 `yaml:"Id"`
	Attributes map[string]interface{} `yaml:"Attributes"`
}
