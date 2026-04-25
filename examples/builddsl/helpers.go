package builddsl

import (
	"fmt"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/lo"
)

func lowerCommands(form compiler.HIRForm) ([]Command, error) {
	commands, err := callsToCommands(form.Calls)
	if err != nil {
		return nil, err
	}
	for _, nested := range form.Forms {
		if nested.Kind != "run" {
			continue
		}
		items, err := callsToCommands(nested.Calls)
		if err != nil {
			return nil, err
		}
		commands = append(commands, items...)
	}
	return commands, nil
}

func callsToCommands(calls []compiler.HIRCall) ([]Command, error) {
	out := make([]Command, 0, len(calls))
	for _, call := range calls {
		values := lo.Map(call.Args, func(arg compiler.HIRArg, _ int) any {
			return arg.Value
		})
		args, err := stringList(values)
		if err != nil {
			return nil, err
		}
		out = append(out, Command{Name: call.Name, Args: args})
	}
	return out, nil
}

func refNames(value any, kind string) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("builddsl: expected list of refs, got %T", value)
	}
	refs := make([]schema.Ref, 0, len(items))
	for _, item := range items {
		ref, ok := item.(schema.Ref)
		if !ok {
			return nil, fmt.Errorf("builddsl: expected ref<%s>, got %T", kind, item)
		}
		if ref.Kind != kind {
			return nil, fmt.Errorf("builddsl: expected ref<%s>, got ref<%s>", kind, ref.Kind)
		}
		refs = append(refs, ref)
	}
	return lo.Map(refs, func(item schema.Ref, _ int) string {
		return item.Name
	}), nil
}

func stringList(value any) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("builddsl: expected list of strings, got %T", value)
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("builddsl: expected string, got %T", item)
		}
		values = append(values, text)
	}
	return lo.Map(values, func(item string, _ int) string {
		return item
	}), nil
}

func validateStringArgs(name string, args []any) error {
	for _, arg := range args {
		if _, ok := arg.(string); !ok {
			return fmt.Errorf("builddsl: action %q expects string arguments, got %T", name, arg)
		}
	}
	return nil
}
