package manifest

import (
	"encoding/json"
	"fmt"
	"os"
)

// ---------------------------------------------------------------------------
// Disk I/O
// ---------------------------------------------------------------------------
// Read and Write are the only two functions that touch the filesystem.
// Everything else in this package works on the in-memory *Manifest.
//
// Unknown-field preservation: Read stores a copy of the original document
// in m.raw, and Write merges typed changes back over that copy. This way
// editing a manifest (e.g. filling in UUIDs) never drops keys we don't
// model, such as "world_template_options", "capabilities", or any future
// fields Mojang might add.

// Read loads the manifest file at path and parses it into a *Manifest.
// The original document is retained (see above) so unknown fields survive
// a later Write.
func Read(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file at '%s': %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("error parsing manifest JSON: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("error parsing manifest JSON: %w", err)
	}
	m.raw = raw

	return &m, nil
}

// Write saves the Manifest to disk at path with 2-space JSON indentation
// plus a trailing newline. Unknown fields captured by Read are preserved;
// known fields always reflect the current struct values.
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

	// Append a trailing newline, per good file convention.
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("error writing manifest to '%s': %w", path, err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Merge helpers (unexported)
// ---------------------------------------------------------------------------
// These overlay the typed struct values onto the raw original document.
// Keys that only exist in the raw document (unknown fields) are kept.
// Nested objects and arrays of objects are merged recursively so extra keys
// inside header/modules/etc. survive as well.

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
