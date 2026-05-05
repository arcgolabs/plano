package lsp

import (
	"cmp"
	"slices"
	"strings"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/diag"
)

const maxCodeActionSuggestions = 5

type codeActionCandidate struct {
	name     string
	distance int
}

func (s Snapshot) CodeActions(rng Range) list.List[CodeAction] {
	out := list.NewList[CodeAction]()
	diagnostics := s.Diagnostics.Values()
	for index := range diagnostics {
		diagnostic := diagnostics[index]
		if !rangeOverlaps(diagnostic.Range, rng) {
			continue
		}
		actions := s.codeActionsForDiagnostic(diagnostic)
		for _, action := range actions.Values() {
			out.Add(action)
		}
	}
	return *out
}

func (s Snapshot) codeActionsForDiagnostic(item Diagnostic) list.List[CodeAction] {
	name, ok := quotedDiagnosticName(item.Message)
	if !ok {
		return list.List[CodeAction]{}
	}
	names := s.codeActionNames(item.Code, item.Range.Start)
	if len(names) == 0 {
		return list.List[CodeAction]{}
	}
	replacementRange := s.codeActionReplacementRange(item, name)
	candidates := rankedCodeActionCandidates(name, names)
	out := list.NewListWithCapacity[CodeAction](len(candidates))
	for index, candidate := range candidates {
		out.Add(CodeAction{
			Title:       `Replace with "` + candidate.name + `"`,
			Kind:        CodeActionQuickFix,
			Diagnostics: *list.NewList(item),
			Edit: WorkspaceEdit{
				Changes: singleTextEdit(s.URI, replacementRange, candidate.name),
			},
			IsPreferred: index == 0,
		})
	}
	return *out
}

func (s Snapshot) codeActionNames(code string, pos Position) []string {
	switch diag.Code(code) {
	case diag.CodeUnknownForm:
		return s.formCodeActionNames()
	case diag.CodeUnknownFunction:
		return s.functionCodeActionNames()
	case diag.CodeUnknownAction:
		return s.actionCodeActionNames()
	case diag.CodeUndefinedName:
		return s.nameCodeActionNames(pos)
	case diag.CodeReadFailure,
		diag.CodeImportCycle,
		diag.CodeDuplicateDefinition,
		diag.CodeTypeMismatch:
		return nil
	default:
		return nil
	}
}

func (s Snapshot) formCodeActionNames() []string {
	index := newCompletionIndex()
	s.addFormCompletions(index)
	return index.items.Keys()
}

func (s Snapshot) functionCodeActionNames() []string {
	index := newCompletionIndex()
	s.addFunctionCompletions(index)
	s.addBuiltinFunctionCompletions(index)
	return index.items.Keys()
}

func (s Snapshot) actionCodeActionNames() []string {
	index := newCompletionIndex()
	s.addActionCompletions(index)
	return index.items.Keys()
}

func (s Snapshot) nameCodeActionNames(pos Position) []string {
	target, ok := s.tokenPos(pos)
	if !ok {
		return nil
	}
	scope := s.completionScope(target)
	index := newCompletionIndex()
	s.addLocalCompletions(index, target, scope)
	s.addConstCompletions(index)
	s.addFunctionCompletions(index)
	s.addSymbolCompletions(index)
	s.addBuiltinFunctionCompletions(index)
	s.addGlobalCompletions(index)
	return index.items.Keys()
}

func (s Snapshot) codeActionReplacementRange(item Diagnostic, name string) Range {
	if useRange, ok := s.unresolvedUseRange(item.Range, name); ok {
		return useRange
	}
	return item.Range
}

func (s Snapshot) unresolvedUseRange(rng Range, name string) (Range, bool) {
	if s.Result.Binding == nil || s.Result.Binding.Uses == nil {
		return Range{}, false
	}
	for _, use := range s.Result.Binding.Uses.Values() {
		if use.Name != name || use.Kind != compiler.UseUnresolved {
			continue
		}
		useRange, ok := s.rangeForSpan(use.Pos, use.End)
		if ok && rangeOverlaps(useRange, rng) {
			return useRange, true
		}
	}
	return Range{}, false
}

func rankedCodeActionCandidates(input string, names []string) []codeActionCandidate {
	seen := set.NewSet[string]()
	candidates := make([]codeActionCandidate, 0, min(len(names), maxCodeActionSuggestions))
	for _, name := range names {
		if name == input || seen.Contains(name) {
			continue
		}
		seen.Add(name)
		distance := levenshteinDistance(input, name)
		if !isCodeActionCandidate(input, name, distance) {
			continue
		}
		candidates = append(candidates, codeActionCandidate{
			name:     name,
			distance: distance,
		})
	}
	slices.SortFunc(candidates, func(left, right codeActionCandidate) int {
		if order := cmp.Compare(left.distance, right.distance); order != 0 {
			return order
		}
		return cmp.Compare(left.name, right.name)
	})
	return candidates[:min(len(candidates), maxCodeActionSuggestions)]
}

func isCodeActionCandidate(input, name string, distance int) bool {
	if strings.HasPrefix(name, input) || strings.HasPrefix(input, name) {
		return true
	}
	limit := max(2, max(len(input), len(name))/3)
	return distance <= limit
}

func quotedDiagnosticName(message string) (string, bool) {
	start := strings.IndexByte(message, '"')
	if start < 0 {
		return "", false
	}
	end := strings.IndexByte(message[start+1:], '"')
	if end < 0 {
		return "", false
	}
	return message[start+1 : start+1+end], true
}

func singleTextEdit(
	uri string,
	rng Range,
	newText string,
) *mapping.OrderedMap[string, list.List[TextEdit]] {
	changes := mapping.NewOrderedMap[string, list.List[TextEdit]]()
	changes.Set(uri, *list.NewList(TextEdit{
		Range:   rng,
		NewText: newText,
	}))
	return changes
}

func rangeOverlaps(left, right Range) bool {
	return comparePosition(left.Start, right.End) <= 0 &&
		comparePosition(right.Start, left.End) <= 0
}

func comparePosition(left, right Position) int {
	if order := cmp.Compare(left.Line, right.Line); order != 0 {
		return order
	}
	return cmp.Compare(left.Character, right.Character)
}

func levenshteinDistance(left, right string) int {
	if left == right {
		return 0
	}
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	previous := make([]int, len(rightRunes)+1)
	current := make([]int, len(rightRunes)+1)
	for index := range previous {
		previous[index] = index
	}
	for leftIndex, leftRune := range leftRunes {
		current[0] = leftIndex + 1
		for rightIndex, rightRune := range rightRunes {
			cost := 1
			if leftRune == rightRune {
				cost = 0
			}
			current[rightIndex+1] = min(
				previous[rightIndex+1]+1,
				current[rightIndex]+1,
				previous[rightIndex]+cost,
			)
		}
		previous, current = current, previous
	}
	return previous[len(rightRunes)]
}
