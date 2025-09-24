package cache

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
