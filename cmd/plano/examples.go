package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/plano/compiler"
	examplebuilddsl "github.com/arcgolabs/plano/examples/builddsl"
	"github.com/arcgolabs/plano/examples/pipelinedsl"
	"github.com/arcgolabs/plano/examples/servicedsl"
)

type exampleSpec struct {
	register    func(*compiler.Compiler) error
	lower       func(*compiler.HIR) (any, error)
	description string
	dir         string
	sample      string
}

func buildExampleRegistry() *mapping.OrderedMap[string, exampleSpec] {
	registry := mapping.NewOrderedMap[string, exampleSpec]()
	registry.Set("builddsl", exampleSpec{
		description: "Build graph with tasks, Go helpers, and run actions",
		dir:         "examples/builddsl",
		sample:      "examples/builddsl/sample.plano",
		register:    examplebuilddsl.Register,
		lower: func(hir *compiler.HIR) (any, error) {
			return examplebuilddsl.Lower(hir)
		},
	})
	registry.Set("pipelinedsl", exampleSpec{
		description: "CI pipeline with stages, dependencies, and runner actions",
		dir:         "examples/pipelinedsl",
		sample:      "examples/pipelinedsl/sample.plano",
		register:    pipelinedsl.Register,
		lower: func(hir *compiler.HIR) (any, error) {
			return pipelinedsl.Lower(hir)
		},
	})
	registry.Set("servicedsl", exampleSpec{
		description: "Service topology with ports, refs, and env maps",
		dir:         "examples/servicedsl",
		sample:      "examples/servicedsl/sample.plano",
		register:    servicedsl.Register,
		lower: func(hir *compiler.HIR) (any, error) {
			return servicedsl.Lower(hir)
		},
	})
	return registry
}

func availableExamples() *mapping.OrderedMap[string, exampleSpec] {
	return buildExampleRegistry()
}

func exampleNames() string {
	return strings.Join(availableExamples().Keys(), ", ")
}

type exampleView struct {
	Name        string   `json:"name"        yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Sample      string   `json:"sample"      yaml:"sample"`
	Samples     []string `json:"samples"     yaml:"samples"`
}

func exampleViews() []exampleView {
	views := make([]exampleView, 0, availableExamples().Len())
	availableExamples().Range(func(name string, spec exampleSpec) bool {
		views = append(views, exampleView{
			Name:        name,
			Description: spec.description,
			Sample:      spec.sample,
			Samples:     exampleSamples(spec),
		})
		return true
	})
	return views
}

func exampleSamples(spec exampleSpec) []string {
	seen := set.NewSet[string]()
	samples := appendSamplePath(nil, seen, spec.sample)
	entries, dir, ok := readExampleDir(spec.dir)
	if !ok {
		return samples
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".plano" {
			continue
		}
		samples = appendSamplePath(samples, seen, filepath.Join(dir, entry.Name()))
	}
	return samples
}

func readExampleDir(dir string) ([]os.DirEntry, string, bool) {
	for _, candidate := range exampleDirCandidates(dir) {
		entries, err := os.ReadDir(candidate)
		if err == nil {
			return entries, dir, true
		}
	}
	return nil, "", false
}

func exampleDirCandidates(dir string) []string {
	if dir == "" {
		return nil
	}
	return []string{dir, filepath.Join("..", "..", dir)}
}

func appendSamplePath(samples []string, seen *set.Set[string], path string) []string {
	if path == "" {
		return samples
	}
	path = filepath.ToSlash(path)
	if seen.Contains(path) {
		return samples
	}
	seen.Add(path)
	return append(samples, path)
}

func newCompilerForExample(name string) (*compiler.Compiler, error) {
	c := compiler.New(compiler.Options{})
	if name == "" {
		return c, nil
	}
	spec, ok := availableExamples().Get(name)
	if !ok {
		return nil, fmt.Errorf("unsupported example %q", name)
	}
	if err := spec.register(c); err != nil {
		return nil, fmt.Errorf("register example %q: %w", name, err)
	}
	return c, nil
}

func lowerDocument(hir *compiler.HIR, name string) (any, error) {
	if name == "" {
		return nil, errors.New("lower requires --example")
	}
	spec, ok := availableExamples().Get(name)
	if !ok {
		return nil, fmt.Errorf("unsupported example %q", name)
	}
	lowered, err := spec.lower(hir)
	if err != nil {
		return nil, fmt.Errorf("lower with %q example: %w", name, err)
	}
	return lowered, nil
}
