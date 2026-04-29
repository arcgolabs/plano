package lsp

import (
	"unicode"
	"unicode/utf8"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/prefix"
)

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
