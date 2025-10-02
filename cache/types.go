package cache

type MxFileMeta struct {
	Hash     string `yaml:"Hash"`
	Path     string `yaml:"Path"`
	DiffType string `yaml:"DiffType"`
}

type MxCacheDiff struct {
	ID       string `yaml:"ID"`
	DiffType string `yaml:"DiffType"`
	Hash     string `yaml:"Hash"`
}

type MxCacheDiffMap map[string]MxCacheDiff

type MxCacheDiffWrapper struct {
	Status string         `yaml:"Status"`
	Data   MxCacheDiffMap `yaml:"Data"`
}

type MxFileListMeta map[string]MxFileMeta
