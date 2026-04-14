// Package scanner extracts Go module dependency information from a target project.
package scanner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Module represents a single Go module entry from `go list -m -json`.
type Module struct {
	Path      string
	Version   string
	Main      bool
	Indirect  bool
	GoVersion string
	Replace   *Module
	Error     *ModuleError
}

// ModuleError holds an error message for a module that failed to load.
type ModuleError struct {
	Err string
}

// Result bundles the scanned modules with their dependency graph.
type Result struct {
	// Modules is the flat list of all modules (main + dependencies).
	Modules []Module
	// Graph maps "module@version" → list of direct dependency "module@version" strings.
	Graph map[string][]string
}

// Scan runs `go list -m -json all` and `go mod graph` inside dir and returns
// the combined result.
func Scan(dir string) (*Result, error) {
	modules, err := listModules(dir)
	if err != nil {
		return nil, err
	}

	graph, err := modGraph(dir)
	if err != nil {
		// Non-fatal: dependency graph is best-effort.
		graph = make(map[string][]string)
	}

	return &Result{Modules: modules, Graph: graph}, nil
}

// listModules runs `go list -m -json all` and decodes the stream of JSON objects.
func listModules(dir string) ([]Module, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list: %w\n%s", err, stderr.String())
	}

	var modules []Module
	dec := json.NewDecoder(&stdout)
	for dec.More() {
		var m Module
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("decoding module info: %w", err)
		}
		modules = append(modules, m)
	}
	return modules, nil
}

// modGraph runs `go mod graph` and returns an adjacency map.
// Each key is "module@version" and the value is a slice of direct dependencies.
func modGraph(dir string) (map[string][]string, error) {
	cmd := exec.Command("go", "mod", "graph")
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go mod graph: %w\n%s", err, stderr.String())
	}

	graph := make(map[string][]string)
	sc := bufio.NewScanner(&stdout)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		from, to := parts[0], parts[1]
		graph[from] = append(graph[from], to)
	}
	return graph, sc.Err()
}
