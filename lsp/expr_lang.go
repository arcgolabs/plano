package lsp

import (
	"reflect"
	"strings"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/compiler"
	"github.com/samber/lo"
)

func (s Snapshot) exprLangHoverAt(pos Position) (Hover, bool) {
	target, src, offset, ok := s.exprLangSourcePosition(pos)
	if !ok {
		return Hover{}, false
	}
	file, ok := s.fileForPath(s.Path)
	if !ok || file == nil {
		return Hover{}, false
	}
	stringTok, ok := exprLangStringTokenAt(file, src, target)
	if !ok {
		return Hover{}, false
	}
	start, end := exprLangStringCompletionBounds(file, src, stringTok, offset)
	if start < 0 || start == end {
		return Hover{}, false
	}
	name := string(src[start:end])
	if hover, ok := s.exprLangVarHover(name, src, start, end); ok {
		return hover, true
	}
	if hover, ok := s.exprLangFuncHover(name, src, start, end); ok {
		return hover, true
	}
	return Hover{}, false
}

func (s Snapshot) exprLangVarHover(name string, src []byte, start, end int) (Hover, bool) {
	if s.compiler == nil {
		return Hover{}, false
	}
	value, ok := s.compiler.ExprVars().Get(name)
	if !ok {
		return Hover{}, false
	}
	return Hover{
		Range:    offsetRange(src, start, end),
		Contents: "```plano\nexpr var " + name + ": " + exprLangTypeDetail(value) + "\n```",
	}, true
}

func (s Snapshot) exprLangFuncHover(name string, src []byte, start, end int) (Hover, bool) {
	if s.compiler == nil {
		return Hover{}, false
	}
	spec, ok := s.compiler.ExprFunctionSpec(name)
	if !ok {
		return Hover{}, false
	}
	return Hover{
		Range:    offsetRange(src, start, end),
		Contents: formatExprFunctionSpec(spec),
	}, true
}

func exprLangVarDetail(value any) string {
	return "expr var " + exprLangTypeDetail(value)
}

func exprLangTypeDetail(value any) string {
	if value == nil {
		return "<nil>"
	}
	return reflect.TypeOf(value).String()
}

func formatExprFunctionSpec(spec compiler.ExprFunctionSpec) string {
	body := "```plano\nexpr fn " + spec.Name + "(" + strings.Join(exprFunctionTypeStrings(spec.Types), ", ") + ")\n```"
	if spec.Docs == "" {
		return body
	}
	return body + "\n\n" + spec.Docs
}

func exprFunctionTypeStrings(types list.List[any]) []string {
	return lo.Map(types.Values(), func(value any, _ int) string {
		return exprLangTypeDetail(value)
	})
}

func offsetRange(src []byte, start, end int) Range {
	return Range{
		Start: positionFromOffset(src, start),
		End:   positionFromOffset(src, end),
	}
}
