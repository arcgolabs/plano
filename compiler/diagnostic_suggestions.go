package compiler

import (
	"cmp"
	"go/token"
	"slices"
	"strings"

	"github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

const maxDiagnosticSuggestions = 5

type diagnosticCandidate struct {
	name     string
	distance int
}

func diagnosticReplacementSuggestions(
	input string,
	names []string,
	pos token.Pos,
	end token.Pos,
) []diag.Suggestion {
	candidates := rankedDiagnosticCandidates(input, names)
	suggestions := make([]diag.Suggestion, 0, len(candidates))
	for _, candidate := range candidates {
		suggestions = append(suggestions, diag.Suggestion{
			Title:       `Replace with "` + candidate.name + `"`,
			Replacement: candidate.name,
			Pos:         pos,
			End:         end,
		})
	}
	return suggestions
}

func rankedDiagnosticCandidates(input string, names []string) []diagnosticCandidate {
	seen := set.NewSet[string]()
	candidates := make([]diagnosticCandidate, 0, min(len(names), maxDiagnosticSuggestions))
	for _, name := range names {
		if name == input || seen.Contains(name) {
			continue
		}
		seen.Add(name)
		distance := levenshteinDistance(input, name)
		if !isDiagnosticCandidate(input, name, distance) {
			continue
		}
		candidates = append(candidates, diagnosticCandidate{
			name:     name,
			distance: distance,
		})
	}
	slices.SortFunc(candidates, func(left, right diagnosticCandidate) int {
		if order := cmp.Compare(left.distance, right.distance); order != 0 {
			return order
		}
		return cmp.Compare(left.name, right.name)
	})
	return candidates[:min(len(candidates), maxDiagnosticSuggestions)]
}

func isDiagnosticCandidate(input, name string, distance int) bool {
	if strings.HasPrefix(name, input) || strings.HasPrefix(input, name) {
		return true
	}
	limit := max(2, max(len(input), len(name))/3)
	return distance <= limit
}

func (c *checker) formSuggestions(name string, pos, end token.Pos) []diag.Suggestion {
	return diagnosticReplacementSuggestions(name, c.compiler.forms.Keys(), pos, end)
}

func (c *checker) functionSuggestions(name string, pos, end token.Pos) []diag.Suggestion {
	names := append([]string{}, c.binding.Functions.Keys()...)
	names = append(names, c.compiler.funcs.Keys()...)
	return diagnosticReplacementSuggestions(name, names, pos, end)
}

func (c *checker) actionSuggestions(name string, pos, end token.Pos) []diag.Suggestion {
	return diagnosticReplacementSuggestions(name, c.compiler.actions.Keys(), pos, end)
}

func (c *checker) fieldSuggestions(spec schema.FormSpec, name string, pos, end token.Pos) []diag.Suggestion {
	return diagnosticReplacementSuggestions(name, spec.Fields.Keys(), pos, end)
}

func (c *checker) nestedFormSuggestions(spec schema.FormSpec, name string, pos, end token.Pos) []diag.Suggestion {
	return diagnosticReplacementSuggestions(name, nestedFormNames(spec), pos, end)
}

func (c *checker) nameSuggestions(name string, scope *checkScope, pos, end token.Pos) []diag.Suggestion {
	return diagnosticReplacementSuggestions(name, c.visibleNameCandidates(scope), pos, end)
}

func (c *checker) visibleNameCandidates(scope *checkScope) []string {
	seen := set.NewSet[string]()
	names := make([]string, 0)
	names = appendScopeNameCandidates(names, seen, scope)
	names = appendUniqueNames(names, seen, c.compiler.globals.Keys()...)
	names = appendUniqueNames(names, seen, c.binding.Consts.Keys()...)
	names = appendUniqueNames(names, seen, c.binding.Symbols.Keys()...)
	return names
}

func appendScopeNameCandidates(names []string, seen *set.Set[string], scope *checkScope) []string {
	for current := scope; current != nil; current = current.parent {
		current.locals.Range(func(name string, _ checkLocalBinding) bool {
			names = appendUniqueName(names, seen, name)
			return true
		})
	}
	return names
}

func appendUniqueNames(names []string, seen *set.Set[string], candidates ...string) []string {
	for _, name := range candidates {
		names = appendUniqueName(names, seen, name)
	}
	return names
}

func appendUniqueName(names []string, seen *set.Set[string], name string) []string {
	if seen.Contains(name) {
		return names
	}
	seen.Add(name)
	return append(names, name)
}

func (s *compileState) formSuggestions(name string, pos, end token.Pos) []diag.Suggestion {
	return diagnosticReplacementSuggestions(name, s.compiler.forms.Keys(), pos, end)
}

func (s *compileState) actionSuggestions(name string, pos, end token.Pos) []diag.Suggestion {
	return diagnosticReplacementSuggestions(name, s.compiler.actions.Keys(), pos, end)
}

func (s *compileState) fieldSuggestions(spec schema.FormSpec, name string, pos, end token.Pos) []diag.Suggestion {
	return diagnosticReplacementSuggestions(name, spec.Fields.Keys(), pos, end)
}

func (s *compileState) nestedFormSuggestions(spec schema.FormSpec, name string, pos, end token.Pos) []diag.Suggestion {
	return diagnosticReplacementSuggestions(name, nestedFormNames(spec), pos, end)
}

func nestedFormNames(spec schema.FormSpec) []string {
	if spec.NestedForms == nil {
		return nil
	}
	names := make([]string, 0, spec.NestedForms.Len())
	spec.NestedForms.Range(func(name string) bool {
		names = append(names, name)
		return true
	})
	return names
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
