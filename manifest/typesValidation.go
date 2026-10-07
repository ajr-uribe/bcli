package manifest

import (
	"encoding/json"
	"fmt"
)

func (m *ModuleType) UnmarshalJSON(data []byte) error {
	var val string
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	switch ModuleType(val) {
	case ModuleTypeResources, ModuleTypeData, ModuleTypeScript:
		*m = ModuleType(val)
		return nil
	default:
		return fmt.Errorf("module.type '%s' not valid (allowed: resources, data, script)", val)
	}
}

func (m *FormatVersion) UnmarshalJSON(data []byte) error {
	var val int
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	switch FormatVersion(val) {
	case FormatVersion1, FormatVersion2, FormatVersion3:
		*m = FormatVersion(val)
		return nil
	default:
		return fmt.Errorf("format_version '%v' not valid (allowed: 1, 2, 3)", val)
	}
}

func (v *Version) UnmarshalJSON(data []byte) error {
	var arr []int
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("version must be an array of three integers, e.g. [1, 0, 0]")
	}
	if len(arr) != 3 {
		return fmt.Errorf("version must have exactly 3 numbers, got %d", len(arr))
	}
	copy(v[:], arr)
	return nil
}

func (d *Dependency) UnmarshalJSON(data []byte) error {
	type Alias Dependency
	var aux Alias

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	hasUUID := aux.UUID != ""
	hasModule := aux.ModuleName != ""

	if hasUUID && hasModule {
		return fmt.Errorf("invalid dependency: has 'uuid' and 'module_name' declared at the same time")
	}
	if !hasUUID && !hasModule {
		return fmt.Errorf("invalid dependency: must have 'uuid' or 'module_name'")
	}

	*d = Dependency(aux)
	return nil
}
