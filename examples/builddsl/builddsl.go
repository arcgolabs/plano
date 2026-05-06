// Package builddsl is an example host DSL built on top of plano.
// It is not part of the stable plano core API.
package builddsl

import (
	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/mo"
)

type Project struct {
	Workspace mo.Option[Workspace]
	Tasks     mapping.OrderedMap[string, Task]
}

type Workspace struct {
	Name        string
	DefaultTask string
}

type Task struct {
	Name     string
	Deps     list.List[string]
	Outputs  list.List[string]
	Commands list.List[Command]
}

type Command struct {
	Name string
	Args list.List[string]
}

func Lower(hir *compiler.HIR) (*Project, error) {
	project := &Project{
		Workspace: mo.None[Workspace](),
	}
	for idx := range hir.Forms.Len() {
		form, _ := hir.Forms.Get(idx)
		if err := applyRootForm(project, form); err != nil {
			return nil, err
		}
	}
	if project.Workspace.IsAbsent() {
		return nil, buildDSLErrorf("workspace form is required")
	}
	return project, nil
}

func applyRootForm(project *Project, form compiler.HIRForm) error {
	if form.Kind == "workspace" {
		return applyWorkspaceForm(project, form)
	}
	if form.Kind == "task" {
		return applyTaskForm(project, form)
	}
	if form.Kind == "go.test" {
		return applyGoTestForm(project, form)
	}
	if form.Kind == "go.binary" {
		return applyGoBinaryForm(project, form)
	}
	return nil
}

func applyWorkspaceForm(project *Project, form compiler.HIRForm) error {
	workspace, err := lowerWorkspace(form)
	if err != nil {
		return err
	}
	if project.Workspace.IsPresent() {
		return buildDSLErrorf("only one workspace form is allowed")
	}
	project.Workspace = mo.Some(workspace)
	return nil
}

func applyTaskForm(project *Project, form compiler.HIRForm) error {
	task, err := lowerTask(form)
	if err != nil {
		return err
	}
	project.Tasks.Set(task.Name, task)
	return nil
}

func applyGoTestForm(project *Project, form compiler.HIRForm) error {
	task, err := lowerGoTestTask(form)
	if err != nil {
		return err
	}
	project.Tasks.Set(task.Name, task)
	return nil
}

func applyGoBinaryForm(project *Project, form compiler.HIRForm) error {
	task, err := lowerGoBinaryTask(form)
	if err != nil {
		return err
	}
	project.Tasks.Set(task.Name, task)
	return nil
}

func lowerWorkspace(form compiler.HIRForm) (Workspace, error) {
	nameValue, ok := form.Field("name")
	if !ok {
		return Workspace{}, buildDSLErrorf("workspace.name is required")
	}
	name, ok := nameValue.Value.(string)
	if !ok {
		return Workspace{}, buildDSLErrorf("workspace.name must be string")
	}
	defaultValue, ok := form.Field("default")
	if !ok {
		return Workspace{}, buildDSLErrorf("workspace.default is required")
	}
	defaultTask, ok := defaultValue.Value.(schema.Ref)
	if !ok || defaultTask.Kind != "task" {
		return Workspace{}, buildDSLErrorf("workspace.default must be ref<task>")
	}
	return Workspace{
		Name:        name,
		DefaultTask: defaultTask.Name,
	}, nil
}

func lowerTask(form compiler.HIRForm) (Task, error) {
	if form.Symbol == nil {
		return Task{}, buildDSLErrorf("task form requires symbol label")
	}

	depsValue, _ := form.Field("deps")
	deps, err := refNames(depsValue.Value, "task")
	if err != nil {
		return Task{}, err
	}
	outputsValue, _ := form.Field("outputs")
	outputs, err := stringList(outputsValue.Value)
	if err != nil {
		return Task{}, err
	}
	commands, err := lowerCommands(form)
	if err != nil {
		return Task{}, err
	}

	return Task{
		Name:     form.Symbol.Name,
		Deps:     deps,
		Outputs:  outputs,
		Commands: commands,
	}, nil
}

func lowerGoTestTask(form compiler.HIRForm) (Task, error) {
	if form.Symbol == nil {
		return Task{}, buildDSLErrorf("go.test form requires symbol label")
	}
	depsValue, _ := form.Field("deps")
	deps, err := refNames(depsValue.Value, "task")
	if err != nil {
		return Task{}, err
	}
	packagesValue, _ := form.Field("packages")
	packages, err := stringList(packagesValue.Value)
	if err != nil {
		return Task{}, err
	}
	return Task{
		Name: form.Symbol.Name,
		Deps: deps,
		Commands: *list.NewList(
			Command{
				Name: "exec",
				Args: *list.NewList(append([]string{"go", "test"}, packages.Values()...)...),
			},
		),
	}, nil
}

func lowerGoBinaryTask(form compiler.HIRForm) (Task, error) {
	if form.Symbol == nil {
		return Task{}, buildDSLErrorf("go.binary form requires symbol label")
	}
	depsValue, _ := form.Field("deps")
	deps, err := refNames(depsValue.Value, "task")
	if err != nil {
		return Task{}, err
	}
	mainValue, ok := form.Field("main")
	if !ok {
		return Task{}, buildDSLErrorf("go.binary.main must be string")
	}
	mainPath, ok := mainValue.Value.(string)
	if !ok {
		return Task{}, buildDSLErrorf("go.binary.main must be string")
	}
	outValue, ok := form.Field("out")
	if !ok {
		return Task{}, buildDSLErrorf("go.binary.out must be string")
	}
	outPath, ok := outValue.Value.(string)
	if !ok {
		return Task{}, buildDSLErrorf("go.binary.out must be string")
	}
	return Task{
		Name:    form.Symbol.Name,
		Deps:    deps,
		Outputs: *list.NewList(outPath),
		Commands: *list.NewList(
			Command{
				Name: "exec",
				Args: *list.NewList("go", "build", "-o", outPath, mainPath),
			},
		),
	}, nil
}
