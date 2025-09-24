package mpr

type MxMetadata struct {
	ProductVersion string     `yaml:"ProductVersion"`
	BuildVersion   string     `yaml:"BuildVersion"`
	Modules        []MxModule `yaml:"Modules"`
}

type MxUnit struct {
	UnitID          string                 `yaml:"UnitID"`
	ContainerID     string                 `yaml:"ContainerID"`
	ContainmentName string                 `yaml:"ContainmentName"`
	Hash            string                 `yaml:"Hash"`
	Contents        map[string]interface{} `yaml:"Contents"`
}

type MxFileMeta struct {
	Hash     string `yaml:"Hash"`
	Path     string `yaml:"Path"`
	diffType string `yaml:"DiffType"`
}

type MxCacheDiff struct {
	ID       string `yaml:"ID"`
	diffType string `yaml:"DiffType"`
}

type MxCacheDiffMap map[string]MxCacheDiff

type MxFileListMeta map[string]MxFileMeta

type MxModule struct {
	Name       string                 `yaml:"Name"`
	ID         string                 `yaml:"ID"`
	Attributes map[string]interface{} `yaml:"Attributes"`
}

type MxFolder struct {
	Name       string                 `yaml:"Name"`
	ID         string                 `yaml:"ID"`
	ParentID   string                 `yaml:"ParentID"`
	Parent     *MxFolder              `yaml:"Parent"`
	Attributes map[string]interface{} `yaml:"Attributes"`
}

type MxMicroflow struct {
	Name       string                 `yaml:"Name"`
	ID         string                 `yaml:"ID"`
	Attributes map[string]interface{} `yaml:"Attributes"`
}

type MxMicroflowEdge struct {
	Type        string                 `yaml:"Type"`
	ID          string                 `yaml:"ID"`
	Origin      string                 `yaml:"Origin"`
	Destination string                 `yaml:"Destination"`
	Attributes  map[string]interface{} `yaml:"Attributes"`
}
type MxMicroflowObject struct {
	Type        string                 `yaml:"Type"`
	ID          string                 `yaml:"ID"`
	Origin      string                 `yaml:"Origin"`
	Destination string                 `yaml:"Destination"`
	Attributes  map[string]interface{} `yaml:"Attributes"`
}

type MxMicroflowNode struct {
	Type       string                 `yaml:"Type"`
	ID         string                 `yaml:"ID"`
	Attributes map[string]interface{} `yaml:"Attributes"`
	Parent     *MxMicroflowNode
	Children   *[]MxMicroflowNode
}

type MxID struct {
	Data    string `yaml:"Data"`
	Subtype int    `yaml:"Subtype"`
}
