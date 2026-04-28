package lsp

import (
	"go/token"
	"strings"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
)

func (s Snapshot) HoverAt(pos Position) (Hover, bool) {
	target, ok := s.tokenPos(pos)
	if !ok {
		return Hover{}, false
	}

	parts, hoverRange, rangeSet := s.hoverData(target)
	if expr, ok := findExprAt(s.Result.Checks, target); ok {
		parts = append(parts, "type: `"+expr.Type.String()+"`")
		if !rangeSet {
			hoverRange, rangeSet = s.rangeForSpan(expr.Pos, expr.End)
		}
	}
	if len(parts) == 0 {
		return Hover{}, false
	}
	if !rangeSet {
		hoverRange = Range{Start: pos, End: pos}
	}
	return Hover{
		Range:    hoverRange,
		Contents: strings.Join(parts, "\n\n"),
	}, true
}

func (s Snapshot) hoverData(target token.Pos) ([]string, Range, bool) {
	if s.Result.Binding == nil {
		return nil, Range{}, false
	}
	if use, ok := findUseAt(s.Result.Binding, target); ok {
		content := s.hoverForUse(use)
		rng, rangeSet := s.rangeForSpan(use.Pos, use.End)
		return nonEmptyParts(content), rng, rangeSet
	}
	content, pos, end, ok := s.hoverForDeclaration(target)
	if !ok {
		return nil, Range{}, false
	}
	rng, rangeSet := s.rangeForSpan(pos, end)
	return nonEmptyParts(content), rng, rangeSet
}

func nonEmptyParts(content string) []string {
	if content == "" {
		return nil
	}
	return []string{content}
}

func (s Snapshot) hoverForDeclaration(target token.Pos) (string, token.Pos, token.Pos, bool) {
	if item, ok := findLocalDeclAt(s.Result.Binding, target); ok {
		return formatLocalBinding(item), item.Pos, item.End, true
	}
	if item, ok := findConstDeclAt(s.Result.Binding, target); ok {
		return formatConstBinding(item), item.Pos, item.End, true
	}
	if item, ok := findFunctionDeclAt(s.Result.Binding, target); ok {
		return formatFunctionBinding(item), item.Pos, item.End, true
	}
	if item, ok := findSymbolDeclAt(s.Result.Binding, target); ok {
		return formatSymbolBinding(item), item.Pos, item.End, true
	}
	return "", token.NoPos, token.NoPos, false
}

func (s Snapshot) hoverForUse(use compiler.NameUse) string {
	switch use.Kind {
	case compiler.UseLocal:
		return s.localHover(use.TargetID)
	case compiler.UseConst:
		return s.constHover(use.TargetID)
	case compiler.UseFunction:
		return s.functionHover(use.TargetID)
	case compiler.UseBuiltinFunction:
		return s.builtinHover(use.TargetID)
	case compiler.UseAction:
		return s.actionHover(use.TargetID)
	case compiler.UseSymbol:
		return s.symbolHover(use.TargetID)
	case compiler.UseGlobal:
		return "```plano\nglobal " + use.TargetID + "\n```"
	case compiler.UseUnresolved:
		return ""
	default:
		return ""
	}
}

func (s Snapshot) localHover(id string) string {
	item, ok := s.Result.Binding.Locals.Get(id)
	if !ok {
		return ""
	}
	return formatLocalBinding(item)
}

func (s Snapshot) constHover(id string) string {
	item, ok := s.Result.Binding.Consts.Get(id)
	if !ok {
		return ""
	}
	return formatConstBinding(item)
}

func (s Snapshot) functionHover(id string) string {
	item, ok := s.Result.Binding.Functions.Get(id)
	if !ok {
		return ""
	}
	return formatFunctionBinding(item)
}

func (s Snapshot) builtinHover(name string) string {
	spec, ok := s.compiler.FunctionSpec(name)
	if !ok {
		return ""
	}
	return formatFunctionSpec(spec)
}

func (s Snapshot) actionHover(name string) string {
	spec, ok := s.compiler.ActionSpec(name)
	if !ok {
		return ""
	}
	return formatActionSpec(spec)
}

func (s Snapshot) symbolHover(id string) string {
	item, ok := s.Result.Binding.Symbols.Get(id)
	if !ok {
		return ""
	}
	return formatSymbolBinding(item)
}

func formatLocalBinding(item compiler.LocalBinding) string {
	return "```plano\n" + string(item.Kind) + " " + item.Name + ": " + item.Type.String() + "\n```"
}

func formatConstBinding(item compiler.ConstBinding) string {
	return "```plano\nconst " + item.Name + ": " + item.Type.String() + "\n```"
}

func formatSymbolBinding(item compiler.Symbol) string {
	return "```plano\nref<" + item.Kind + "> " + item.Name + "\n```"
}

func formatFunctionBinding(item compiler.FunctionBinding) string {
	return "```plano\nfn " + item.Name + "(" + strings.Join(paramStrings(item.Params), ", ") + "): " + item.Result.String() + "\n```"
}

func formatFunctionSpec(spec schema.FunctionSpec) string {
	body := "```plano\nfn " + spec.Name + "(" + strings.Join(typeStrings(spec.ParamTypes, spec.VariadicType), ", ") + "): " + spec.Result.String() + "\n```"
	if spec.Docs == "" {
		return body
	}
	return body + "\n\n" + spec.Docs
}

func formatActionSpec(spec compiler.ActionSpec) string {
	body := "```plano\naction " + spec.Name + "(" + strings.Join(typeStrings(spec.ArgTypes, spec.VariadicType), ", ") + ")\n```"
	if spec.Docs == "" {
		return body
	}
	return body + "\n\n" + spec.Docs
}

func paramStrings(params []compiler.ParamBinding) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, param.Name+": "+param.Type.String())
	}
	return out
}

func typeStrings(items []schema.Type, variadic schema.Type) []string {
	out := make([]string, 0, len(items)+1)
	for _, item := range items {
		out = append(out, item.String())
	}
	if variadic != nil {
		out = append(out, "..."+variadic.String())
	}
	return out
}
