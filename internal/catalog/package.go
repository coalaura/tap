package catalog

type Package struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Source      string            `yaml:"source"`
	Repository  string            `yaml:"repository"`
	Prefix      string            `yaml:"prefix"`
	Binary      string            `yaml:"binary"`
	Companions  []string          `yaml:"companions"`
	Args        []string          `yaml:"args"`
	Asset       string            `yaml:"asset"`
	Checksums   string            `yaml:"checksums"`
	Platforms   map[string]Target `yaml:"platforms"`
}

type Target struct {
	Asset      string `yaml:"asset"`
	Executable string `yaml:"executable"`
	Path       string `yaml:"path"`
}
