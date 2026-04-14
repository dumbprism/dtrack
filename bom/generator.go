// Package bom converts scanned module data into a CycloneDX Software Bill of Materials.
package bom

import (
	"fmt"
	"io"
	"strings"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"

	"github.com/arhantkotamraju/dtrack/scanner"
)

// Generate builds a CycloneDX BOM from a scanner.Result.
func Generate(result *scanner.Result) *cdx.BOM {
	bom := cdx.NewBOM()

	now := time.Now().UTC()
	bom.Metadata = &cdx.Metadata{
		Timestamp: now.Format(time.RFC3339),
		Tools: &cdx.ToolsChoice{
			Components: &[]cdx.Component{
				{
					Type:    cdx.ComponentTypeApplication,
					Name:    "dtrack",
					Version: "0.1.0",
					Author:  "arhantkotamraju",
				},
			},
		},
	}

	// Index modules by their key used in the dependency graph ("path@version").
	modByKey := make(map[string]*scanner.Module, len(result.Modules))
	var mainMod *scanner.Module

	for i := range result.Modules {
		m := &result.Modules[i]
		if m.Main {
			mainMod = m
			continue
		}
		if m.Version == "" || m.Error != nil {
			continue
		}
		modByKey[m.Path+"@"+m.Version] = m
	}

	// Set main component metadata.
	if mainMod != nil {
		bom.Metadata.Component = &cdx.Component{
			BOMRef:  mainMod.Path,
			Type:    cdx.ComponentTypeApplication,
			Name:    mainMod.Path,
			Version: mainMod.GoVersion,
		}
	}

	// Build components slice.
	var components []cdx.Component
	for key, m := range modByKey {
		props := []cdx.Property{
			{Name: "go:indirect", Value: fmt.Sprintf("%v", m.Indirect)},
		}

		// If the module is replaced, record the replacement.
		if m.Replace != nil {
			props = append(props, cdx.Property{
				Name:  "go:replace",
				Value: m.Replace.Path + "@" + m.Replace.Version,
			})
		}

		comp := cdx.Component{
			BOMRef:     key,
			Type:       cdx.ComponentTypeLibrary,
			Name:       m.Path,
			Version:    m.Version,
			PackageURL: purl(m),
			Properties: &props,
		}
		components = append(components, comp)
	}
	bom.Components = &components

	// Build dependency relationships.
	var deps []cdx.Dependency
	for from, tos := range result.Graph {
		// Use the main module path as a key when the graph entry has no version.
		fromRef := graphKeyToBOMRef(from, mainMod)
		var dependsOn []string
		for _, to := range tos {
			if _, ok := modByKey[to]; ok {
				dependsOn = append(dependsOn, to)
			}
		}
		if len(dependsOn) > 0 {
			deps = append(deps, cdx.Dependency{
				Ref:          fromRef,
				Dependencies: &dependsOn,
			})
		}
	}
	if len(deps) > 0 {
		bom.Dependencies = &deps
	}

	return bom
}

// Encode writes the BOM to w in the requested format ("json" or "xml").
func Encode(b *cdx.BOM, w io.Writer, format string) error {
	var ff cdx.BOMFileFormat
	switch strings.ToLower(format) {
	case "json":
		ff = cdx.BOMFileFormatJSON
	case "xml":
		ff = cdx.BOMFileFormatXML
	default:
		return fmt.Errorf("unsupported format %q: choose json or xml", format)
	}

	enc := cdx.NewBOMEncoder(w, ff)
	enc.SetPretty(true)
	return enc.Encode(b)
}

// purl returns a Package URL for a Go module.
// Format: pkg:golang/<module-path>@<version>
func purl(m *scanner.Module) string {
	path := m.Path
	version := m.Version
	if m.Replace != nil {
		path = m.Replace.Path
		version = m.Replace.Version
	}
	return fmt.Sprintf("pkg:golang/%s@%s", path, version)
}

// graphKeyToBOMRef normalises a graph key (which may be bare "module/path" for
// the main module) to the BOM ref we assigned.
func graphKeyToBOMRef(key string, main *scanner.Module) string {
	if main != nil && key == main.Path {
		return main.Path
	}
	return key
}
