package builddsl

import (
	"fmt"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/lo"
)

func lowerCommands(form compiler.HIRForm) (list.List[Command], error) {
	commands, err := callsToCommands(form.Calls)
	if err != nil {
		return list.List[Command]{}, err
	}
	for idx := range form.Forms.Len() {
		nested, _ := form.Forms.Get(idx)
		if nested.Kind != "run" {
			continue
		}
		items, err := callsToCommands(nested.Calls)
		if err != nil {
			return list.List[Command]{}, err
		}
		commands.Merge(&items)
	}
	return commands, nil
}

func callsToCommands(calls list.List[compiler.HIRCall]) (list.List[Command], error) {
	out := list.NewListWithCapacity[Command](calls.Len())
	for idx := range calls.Len() {
		call, _ := calls.Get(idx)
		values := lo.Map(call.Args.Values(), func(arg compiler.HIRArg, _ int) any {
			return arg.Value
		})
		args, err := stringList(values)
		if err != nil {
			return list.List[Command]{}, err
		}
		out.Add(Command{Name: call.Name, Args: args})
	}
	return *out, nil
}

func refNames(value any, kind string) (list.List[string], error) {
	items, ok := value.([]any)
	if !ok {
		return list.List[string]{}, fmt.Errorf("builddsl: expected list of refs, got %T", value)
	}
	refs := make([]schema.Ref, 0, len(items))
	for _, item := range items {
		ref, ok := item.(schema.Ref)
		if !ok {
			return list.List[string]{}, fmt.Errorf("builddsl: expected ref<%s>, got %T", kind, item)
		}
		if ref.Kind != kind {
			return list.List[string]{}, fmt.Errorf("builddsl: expected ref<%s>, got ref<%s>", kind, ref.Kind)
		}
		refs = append(refs, ref)
	}
	return *list.NewList(lo.Map(refs, func(item schema.Ref, _ int) string {
		return item.Name
	})...), nil
}

func stringList(value any) (list.List[string], error) {
	items, ok := value.([]any)
	if !ok {
		return list.List[string]{}, fmt.Errorf("builddsl: expected list of strings, got %T", value)
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return list.List[string]{}, fmt.Errorf("builddsl: expected string, got %T", item)
		}
		values = append(values, text)
	}
	return *list.NewList(lo.Map(values, func(item string, _ int) string {
		return item
	})...), nil
}

func validateStringArgs(name string, args list.List[any]) error {
	for _, arg := range args.Values() {
		if _, ok := arg.(string); !ok {
			return fmt.Errorf("builddsl: action %q expects string arguments, got %T", name, arg)
		}
	}
	return nil
}
