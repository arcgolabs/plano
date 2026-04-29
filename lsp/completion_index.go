package lsp

import (
	"go/token"
	"unicode"
	"unicode/utf8"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/prefix"
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

func (s Snapshot) addFieldCompletions(index *completionIndex, formKind string) {
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
	if spec.NestedForms != nil {
		spec.NestedForms.Range(func(name string) bool {
			index.add(s.formCompletionItem(name))
			return true
		})
	}
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

func (s Snapshot) addKeywordCompletions(index *completionIndex) {
	for _, item := range keywordCompletions {
		index.add(item)
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

func formCompletionItem(spec schema.FormSpec) CompletionItem {
	return CompletionItem{
		Label:         spec.Name,
		Kind:          CompletionForm,
		Detail:        "form",
		Documentation: formatFormSpec(spec),
	}
}

func formatFormSpec(spec schema.FormSpec) string {
	body := "```plano\n" + spec.Name + " { ... }\n```"
	if spec.Docs == "" {
		return body
	}
	return body + "\n\n" + spec.Docs
}

func formatFieldSpec(formKind string, field schema.FieldSpec) string {
	typ := "any"
	if field.Type != nil {
		typ = field.Type.String()
	}
	body := "```plano\n" + formKind + "." + field.Name + ": " + typ + "\n```"
	if field.Docs == "" {
		return body
	}
	return body + "\n\n" + field.Docs
}

func detailWithType(label string, typ schema.Type) string {
	if typ == nil {
		return label
	}
	return label + " " + typ.String()
}

func newCompletionIndex() *completionIndex {
	return &completionIndex{
		items: mapping.NewOrderedMap[string, CompletionItem](),
		trie:  prefix.NewTrie[CompletionItem](),
	}
}

func (i *completionIndex) add(item CompletionItem) {
	if i == nil || item.Label == "" {
		return
	}
	if _, exists := i.items.Get(item.Label); exists {
		return
	}
	i.items.Set(item.Label, item)
	i.trie.Put(item.Label, item)
}

func (i *completionIndex) match(query string) list.List[CompletionItem] {
	if i == nil {
		return list.List[CompletionItem]{}
	}
	entries := i.trie.EntriesWithPrefix(query)
	if len(entries) == 0 {
		return list.List[CompletionItem]{}
	}
	items := list.NewListWithCapacity[CompletionItem](len(entries))
	for _, entry := range entries {
		items.Add(entry.Value)
	}
	return *items
}

func completionBounds(src []byte, offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}

	start := offset
	for start > 0 {
		r, size := utf8.DecodeLastRune(src[:start])
		if !isCompletionRune(r) {
			break
		}
		start -= size
	}

	end := offset
	for end < len(src) {
		r, size := utf8.DecodeRune(src[end:])
		if !isCompletionRune(r) {
			break
		}
		end += size
	}
	return start, end
}

func isCompletionRune(r rune) bool {
	return r == '.' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
