package manifest

type (
	ModuleType    string
	FormatVersion int
	Version       [3]int
	ProductType   string
)

const (
	ModuleTypeResources ModuleType = "resources"
	ModuleTypeData      ModuleType = "data"
	ModuleTypeScript    ModuleType = "script"

	FormatVersion1 FormatVersion = 1
	FormatVersion2 FormatVersion = 2
	FormatVersion3 FormatVersion = 3

	ProductTypeAddon ProductType = "addon"
)

type Header struct {
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	UUID             string  `json:"uuid"`
	Version          Version `json:"version"`
	MinEngineVersion Version `json:"min_engine_version"`
}

type Module struct {
	Type        ModuleType `json:"type"`
	Description string     `json:"description,omitempty"`
	UUID        string     `json:"uuid"`
	Version     Version    `json:"version"`
}

type Dependency struct {
	UUID       string `json:"uuid,omitempty"`
	ModuleName string `json:"module_name,omitempty"`
	Version    any    `json:"version"`
}

type Metadata struct {
	Authors       []string            `json:"authors,omitempty"`
	License       string              `json:"license,omitempty"`
	URL           string              `json:"url,omitempty"`
	ProductType   ProductType         `json:"product_type,omitempty"`
	GeneratedWith map[string][]string `json:"generated_with,omitempty"`
}

type Manifest struct {
	FormatVersion FormatVersion `json:"format_version"`
	Header        Header        `json:"header"`
	Modules       []Module      `json:"modules"`
	Dependencies  []Dependency  `json:"dependencies,omitempty"`
	Metadata      Metadata      `json:"metadata,omitempty"`
	// raw holds the original decoded document so unknown fields can be
	// preserved when writing the manifest back to disk.
	raw map[string]any
}
