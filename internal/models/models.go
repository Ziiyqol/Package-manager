package models

// CreateConfig представляет структуру файла packet.json.
type CreateConfig struct {
	Name    string         `json:"name" yaml:"name"`
	Ver     string         `json:"ver" yaml:"ver"`
	Targets []TargetConfig `json:"targets" yaml:"targets"`
}

// TargetConfig представляет элемент в массиве `targets`.
type TargetConfig struct {
	Path    string `json:"path" yaml:"path"`
	Exclude string `json:"exclude,omitempty" yaml:"exclude,omitempty"`
}

// UpdateConfig представляет структуру файла packages.json.
type UpdateConfig struct {
	Packages []Package `json:"packages" yaml:"packages"`
}

// Package представляет элемент в массиве `packages`.
type Package struct {
	Name string `json:"name" yaml:"name"`
	Ver  string `json:"ver,omitempty" yaml:"ver,omitempty"`
}
