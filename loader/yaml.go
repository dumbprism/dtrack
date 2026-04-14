// Package loader reads an existing CycloneDX SBOM from a YAML file.
package loader

import (
	"fmt"
	"os"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

// FromYAML parses a CycloneDX BOM from a YAML file at path.
// cyclonedx-go has no native YAML decoder, so the file is converted
// YAML → JSON in memory before being handed to the standard JSON decoder.
func FromYAML(path string) (*cdx.BOM, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	jsonBytes, err := yamlToJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("converting %s to JSON: %w", path, err)
	}

	bom := new(cdx.BOM)
	if err := cdx.NewBOMDecoder(newBytesReader(jsonBytes), cdx.BOMFileFormatJSON).Decode(bom); err != nil {
		return nil, fmt.Errorf("decoding BOM from %s: %w", path, err)
	}
	return bom, nil
}
