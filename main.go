// dtrack - Go dependency tracker that produces CycloneDX SBOMs.
//
// Usage:
//
//	dtrack [flags]
//
// Flags:
//
//	-dir    Path to the Go project to scan (default: current directory)
//	-input  Path to an existing CycloneDX YAML SBOM to load instead of scanning
//	-format Output format: json or xml (default: json)
//	-output Path to write the SBOM; omit to print to stdout
package main

import (
	"flag"
	"fmt"
	"os"

	cdx "github.com/CycloneDX/cyclonedx-go"

	"github.com/arhantkotamraju/dtrack/bom"
	"github.com/arhantkotamraju/dtrack/loader"
	"github.com/arhantkotamraju/dtrack/scanner"
)

const version = "0.1.0"

func main() {
	var (
		dir    = flag.String("dir", ".", "Go project directory to scan")
		input  = flag.String("input", "", "Existing CycloneDX YAML file to load (skips scanning)")
		format = flag.String("format", "json", "Output format: json or xml")
		output = flag.String("output", "", "Output file (default: stdout)")
		ver    = flag.Bool("version", false, "Print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "dtrack %s — CycloneDX dependency tracker for Go modules\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: dtrack [flags]\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  dtrack -dir ./myproject -format json -output sbom.json\n")
		fmt.Fprintf(os.Stderr, "  dtrack -dir ./myproject -format xml  -output sbom.xml\n")
		fmt.Fprintf(os.Stderr, "  dtrack -input sbom.cdx.yaml -format json\n")
		fmt.Fprintf(os.Stderr, "  dtrack -input sbom.cdx.yaml -format xml  -output sbom.xml\n")
	}
	flag.Parse()

	if *ver {
		fmt.Println("dtrack", version)
		return
	}

	var b *cdx.BOM

	if *input != "" {
		// Load an existing CycloneDX YAML SBOM.
		loaded, err := loader.FromYAML(*input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load error: %v\n", err)
			os.Exit(1)
		}
		b = loaded
		printSummary(b)
	} else {
		// Scan a Go project and generate the BOM.
		result, err := scanner.Scan(*dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
			os.Exit(1)
		}
		b = bom.Generate(result)
	}

	w := os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot create output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}

	if err := bom.Encode(b, w, *format); err != nil {
		fmt.Fprintf(os.Stderr, "encode error: %v\n", err)
		os.Exit(1)
	}

	if *output != "" {
		fmt.Fprintf(os.Stderr, "SBOM written to %s\n", *output)
	}
}

// printSummary logs a human-readable overview of a loaded BOM to stderr.
func printSummary(b *cdx.BOM) {
	name := "<unknown>"
	if b.Metadata != nil && b.Metadata.Component != nil {
		name = b.Metadata.Component.Name
	}
	count := 0
	if b.Components != nil {
		count = len(*b.Components)
	}
	fmt.Fprintf(os.Stderr, "Loaded SBOM: %s  spec=%s  components=%d\n", name, b.SpecVersion, count)
}
