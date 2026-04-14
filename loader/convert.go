package loader

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// yamlToJSON converts a YAML document to its JSON equivalent.
// It round-trips through interface{} so that YAML maps become JSON objects.
func yamlToJSON(src []byte) ([]byte, error) {
	var doc interface{}
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	// yaml.v3 decodes maps as map[string]interface{} when keys are strings,
	// but nested maps may be map[interface{}]interface{} — normalise them.
	doc = normalise(doc)

	out, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %w", err)
	}
	return out, nil
}

// normalise recursively converts map[interface{}]interface{} → map[string]interface{}
// so encoding/json can serialise the tree.
func normalise(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, v2 := range val {
			m[fmt.Sprintf("%v", k)] = normalise(v2)
		}
		return m
	case map[string]interface{}:
		for k, v2 := range val {
			val[k] = normalise(v2)
		}
		return val
	case []interface{}:
		for i, v2 := range val {
			val[i] = normalise(v2)
		}
		return val
	default:
		return v
	}
}

// newBytesReader wraps a byte slice in an io.Reader.
func newBytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
