package manifest

import (
	"encoding/json"
	"fmt"
	"os"
)

// Write saves the Manifest struct to disk at the specified path with 2-space JSON indentation.
// Unknown fields from the original document (captured by Read) are preserved.
func Write(path string, m *Manifest) error {
	structData, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("error encoding manifest to JSON: %w", err)
	}

	var structMap map[string]any
	if err := json.Unmarshal(structData, &structMap); err != nil {
		return fmt.Errorf("error encoding manifest to JSON: %w", err)
	}

	merged := structMap
	if m.raw != nil {
		merged = mergeMaps(m.raw, structMap)
	}

	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding manifest to JSON: %w", err)
	}

	// Append a trailing newline, per good file convention
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("error writing manifest to '%s': %w", path, err)
	}

	return nil
}

// mergeMaps overlays values from overlay onto base. Keys that only exist in
// base are kept (unknown fields are preserved). Nested objects and arrays of
// objects are merged recursively so extra keys inside header/modules/etc.
// survive as well.
func mergeMaps(base, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		if bv, ok := out[k]; ok {
			if bm, ok := bv.(map[string]any); ok {
				if ov, ok := v.(map[string]any); ok {
					out[k] = mergeMaps(bm, ov)
					continue
				}
			}
			if ba, ok := bv.([]any); ok {
				if oa, ok := v.([]any); ok {
					out[k] = mergeArrays(ba, oa)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func mergeArrays(base, overlay []any) []any {
	n := len(base)
	if len(overlay) > n {
		n = len(overlay)
	}
	out := make([]any, n)
	for i := range out {
		var b, o any
		if i < len(base) {
			b = base[i]
		}
		if i < len(overlay) {
			o = overlay[i]
		}
		if bm, ok := b.(map[string]any); ok {
			if om, ok := o.(map[string]any); ok {
				out[i] = mergeMaps(bm, om)
				continue
			}
		}
		if o != nil {
			out[i] = o
		} else {
			out[i] = b
		}
	}
	return out
}
