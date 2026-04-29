package lsp

import (
	"go/token"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
)

func (s Snapshot) addLocalCompletions(index *completionIndex, target token.Pos, scope completionScope) {
	if s.Result.Binding == nil || s.Result.Binding.Locals == nil {
		return
	}
	s.Result.Binding.Locals.Range(func(_ string, item compiler.LocalBinding) bool {
		if !scope.ids.Contains(item.ScopeID) || item.Pos > target {
			return true
		}
		index.add(CompletionItem{
			Label:         item.Name,
			Kind:          CompletionLocal,
			Detail:        detailWithType(string(item.Kind), item.Type),
			Documentation: formatLocalBinding(item),
		})
		return true
	})
}

func (s Snapshot) addConstCompletions(index *completionIndex) {
	if s.Result.Binding == nil || s.Result.Binding.Consts == nil {
		return
	}
	s.Result.Binding.Consts.Range(func(_ string, item compiler.ConstBinding) bool {
		index.add(CompletionItem{
			Label:         item.Name,
			Kind:          CompletionConst,
			Detail:        detailWithType("const", item.Type),
			Documentation: formatConstBinding(item),
		})
		return true
	})
}

func (s Snapshot) addFunctionCompletions(index *completionIndex) {
	if s.Result.Binding == nil || s.Result.Binding.Functions == nil {
		return
	}
	s.Result.Binding.Functions.Range(func(_ string, item compiler.FunctionBinding) bool {
		index.add(CompletionItem{
			Label:         item.Name,
			Kind:          CompletionFunction,
			Detail:        detailWithType("fn", item.Result),
			Documentation: formatFunctionBinding(item),
		})
		return true
	})
}

func (s Snapshot) addSymbolCompletions(index *completionIndex) {
	if s.Result.Binding == nil || s.Result.Binding.Symbols == nil {
		return
	}
	s.Result.Binding.Symbols.Range(func(_ string, item compiler.Symbol) bool {
		index.add(CompletionItem{
			Label:         item.Name,
			Kind:          CompletionSymbol,
			Detail:        "ref<" + item.Kind + ">",
			Documentation: formatSymbolBinding(item),
		})
		return true
	})
}

func (s Snapshot) addFieldCompletions(index *completionIndex, formKind string, includeNested bool) {
	if formKind == "" || s.compiler == nil {
		return
	}
	spec, ok := s.compiler.FormSpec(formKind)
	if !ok {
		return
	}
	if spec.Fields != nil {
		spec.Fields.Range(func(_ string, field schema.FieldSpec) bool {
			index.add(CompletionItem{
				Label:         field.Name,
				Kind:          CompletionField,
				Detail:        detailWithType("field", field.Type),
				Documentation: formatFieldSpec(formKind, field),
			})
			return true
		})
	}
	if includeNested {
		s.addNestedFormCompletions(index, formKind)
	}
}

func (s Snapshot) addNestedFormCompletions(index *completionIndex, formKind string) {
	if formKind == "" || s.compiler == nil {
		return
	}
	spec, ok := s.compiler.FormSpec(formKind)
	if !ok || spec.NestedForms == nil {
		return
	}
	spec.NestedForms.Range(func(name string) bool {
		index.add(s.formCompletionItem(name))
		return true
	})
}

func (s Snapshot) addFormCompletions(index *completionIndex) {
	if s.compiler == nil {
		return
	}
	s.compiler.FormSpecs().Range(func(_ string, spec schema.FormSpec) bool {
		index.add(formCompletionItem(spec))
		return true
	})
}

func (s Snapshot) addBuiltinFunctionCompletions(index *completionIndex) {
	if s.compiler == nil {
		return
	}
	s.compiler.FunctionSpecs().Range(func(_ string, spec schema.FunctionSpec) bool {
		index.add(CompletionItem{
			Label:         spec.Name,
			Kind:          CompletionFunction,
			Detail:        detailWithType("fn", spec.Result),
			Documentation: formatFunctionSpec(spec),
		})
		return true
	})
}

func (s Snapshot) addActionCompletions(index *completionIndex) {
	if s.compiler == nil {
		return
	}
	s.compiler.ActionSpecs().Range(func(_ string, spec compiler.ActionSpec) bool {
		index.add(CompletionItem{
			Label:         spec.Name,
			Kind:          CompletionAction,
			Detail:        "action",
			Documentation: formatActionSpec(spec),
		})
		return true
	})
}

func (s Snapshot) addGlobalCompletions(index *completionIndex) {
	if s.compiler == nil {
		return
	}
	s.compiler.Globals().Range(func(name string, _ any) bool {
		index.add(CompletionItem{
			Label:  name,
			Kind:   CompletionGlobal,
			Detail: "global",
		})
		return true
	})
}

func (s Snapshot) addTopLevelKeywords(index *completionIndex) {
	for _, item := range keywordCompletions {
		switch item.Label {
		case "import", "const", "fn":
			index.add(item)
		}
	}
}

func (s Snapshot) addExpressionKeywords(index *completionIndex) {
	for _, item := range keywordCompletions {
		switch item.Label {
		case "true", "false", "null":
			index.add(item)
		}
	}
}

func (s Snapshot) addScriptKeywords(index *completionIndex) {
	for _, item := range keywordCompletions {
		switch item.Label {
		case "let", "if", "for", "return", "break", "continue":
			index.add(item)
		}
	}
}

func (s Snapshot) formCompletionItem(name string) CompletionItem {
	if s.compiler == nil {
		return CompletionItem{
			Label:  name,
			Kind:   CompletionForm,
			Detail: "form",
		}
	}
	spec, ok := s.compiler.FormSpec(name)
	if !ok {
		return CompletionItem{
			Label:  name,
			Kind:   CompletionForm,
			Detail: "form",
		}
	}
	return formCompletionItem(spec)
}
