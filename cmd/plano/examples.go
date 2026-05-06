package main

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/arcgolabs/collectionx/list"
)

//go:embed samples/*.plano
var embeddedSamples embed.FS

type exampleSpec struct {
	name        string
	description string
	path        string
}

type exampleView struct {
	Name        string `json:"name"        yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Path        string `json:"path"        yaml:"path"`
}

type exampleFileView struct {
	Name        string `json:"name"        yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Path        string `json:"path"        yaml:"path"`
	Content     string `json:"content"     yaml:"content"`
}

func availableExamples() list.List[exampleSpec] {
	return *list.NewList(
		exampleSpec{
			name:        "basic",
			description: "Schema-free snippet that works with core CLI compile/check commands",
			path:        "samples/basic.plano",
		},
		exampleSpec{
			name:        "build",
			description: "Build graph snippet with tasks and run actions",
			path:        "samples/build.plano",
		},
		exampleSpec{
			name:        "pipeline",
			description: "CI pipeline snippet with stages and dependencies",
			path:        "samples/pipeline.plano",
		},
		exampleSpec{
			name:        "service",
			description: "Service topology snippet with ports and env maps",
			path:        "samples/service.plano",
		},
	)
}

func exampleViews() *list.List[exampleView] {
	examples := availableExamples()
	views := list.NewListWithCapacity[exampleView](examples.Len())
	for index := range examples.Len() {
		spec, _ := examples.Get(index)
		views.Add(exampleView{
			Name:        spec.name,
			Description: spec.description,
			Path:        spec.path,
		})
	}
	return views
}

func exampleFile(name string) (exampleFileView, error) {
	spec, ok := exampleByName(name)
	if !ok {
		return exampleFileView{}, fmt.Errorf("unknown sample %q", name)
	}
	data, err := fs.ReadFile(embeddedSamples, spec.path)
	if err != nil {
		return exampleFileView{}, fmt.Errorf("read embedded sample %q: %w", name, err)
	}
	return exampleFileView{
		Name:        spec.name,
		Description: spec.description,
		Path:        spec.path,
		Content:     string(data),
	}, nil
}

func exampleByName(name string) (exampleSpec, bool) {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	examples := availableExamples()
	for index := range examples.Len() {
		spec, _ := examples.Get(index)
		if spec.name == name || filepath.Base(spec.path) == name {
			return spec, true
		}
	}
	return exampleSpec{}, false
}
