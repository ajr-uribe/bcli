package manifest

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

// UnmarshalJSON accepts both the array form ([1, 0, 0]) and the
// string form ("1.0.0"). Note that it is always written back as an array.
func (v *Version) UnmarshalJSON(data []byte) error {
	var arr []int
	if err := json.Unmarshal(data, &arr); err == nil {
		if len(arr) != 3 {
			return fmt.Errorf("version must have exactly 3 numbers, got %d", len(arr))
		}

		for _, n := range arr {
			if n < 0 {
				return fmt.Errorf("version numbers cannot be negative, got %d", n)
			}
		}

		copy(v[:], arr)

		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf(`version must be an array of three integers (e.g. [1, 0, 0]) or a string (e.g. "1.0.0")`)
	}

	parts := strings.Split(str, ".")
	if len(parts) != 3 {
		return fmt.Errorf("version string %q must look like 1.0.0", str)
	}

	var parsed Version

	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return fmt.Errorf("version string %q must look like 1.0.0", str)
		}

		parsed[i] = n
	}

	*v = parsed

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

func (pt *ProductType) UnmarshalJSON(b []byte) error {
	var val string
	if err := json.Unmarshal(b, &val); err != nil {
		return fmt.Errorf("product_type must be a string")
	}

	if val != string(ProductTypeAddon) {
		return fmt.Errorf("product_type '%s' is invalid (allowed: 'addon')", val)
	}

	*pt = ProductType(val)
	return nil
}

// UnmarshalJSON only accepts the array form ([1, 26, 0]); strings are rejected.
func (v *EngineVersion) UnmarshalJSON(data []byte) error {
	var arr []int
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("min_engine_version must be an array of three integers, e.g. [1, 26, 0]")
	}

	if len(arr) != 3 {
		return fmt.Errorf("min_engine_version must have exactly 3 numbers, got %d", len(arr))
	}

	for _, n := range arr {
		if n < 0 {
			return fmt.Errorf("min_engine_version numbers cannot be negative, got %d", n)
		}
	}

	copy(v[:], arr)

	return nil
}
