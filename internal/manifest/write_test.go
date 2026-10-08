package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestWritePreservesUnknownFields ensures Read+Write keeps fields the
// typed struct does not know about, at the top level and nested.
func TestWritePreservesUnknownFields(t *testing.T) {
	input := `{
  "format_version": 2,
  "header": {
    "name": "pack",
    "description": "desc",
    "uuid": "458f6e14-40fd-4109-99aa-2ed058eff1fd",
    "version": [1, 0, 0],
    "min_engine_version": [1, 26, 0],
    "custom_header_field": "keep-me"
  },
  "modules": [
    {
      "type": "data",
      "uuid": "efa63950-602a-4c10-af00-37ec0be8f346",
      "version": [1, 0, 0],
      "capabilities": ["chemistry"]
    }
  ],
  "world_template_options": { "template_uuid": "abc" }
}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, m); err != nil {
		t.Fatal(err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var want, got map[string]any
	if err := json.Unmarshal([]byte(input), &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("round-trip changed the document.\nwant: %v\ngot:  %v", want, got)
	}
}

func TestWriteMergesKnownFieldUpdates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	input := `{
  "format_version": 2,
  "header": {"name": "pack", "description": "d", "uuid": "", "version": [1, 0, 0]},
  "modules": [{"type": "data", "uuid": "", "version": [1, 0, 0]}],
  "extra": true
}
`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	EnsureUUIDs(m)
	if err := Write(path, m); err != nil {
		t.Fatal(err)
	}

	out, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["extra"] != true {
		t.Error("expected unknown top-level field to survive")
	}
	header := doc["header"].(map[string]any)
	if header["uuid"] == "" {
		t.Error("expected header uuid to be updated")
	}
}
