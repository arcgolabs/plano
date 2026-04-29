package lsp

import (
	"sync"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/plano/compiler"
)

type snapshotQueryCache struct {
	documentSymbolsOnce sync.Once
	documentSymbols     list.List[DocumentSymbol]

	referencesMu      sync.RWMutex
	referenceComputed *set.Set[string]
	referenceUses     *mapping.MultiMap[string, Location]
	referenceWithDecl *mapping.MultiMap[string, Location]
}

func newSnapshotQueryCache() *snapshotQueryCache {
	return &snapshotQueryCache{
		referenceComputed: set.NewSet[string](),
		referenceUses:     mapping.NewMultiMap[string, Location](),
		referenceWithDecl: mapping.NewMultiMap[string, Location](),
	}
}

func (s Snapshot) cachedDocumentSymbols() list.List[DocumentSymbol] {
	if s.queries == nil {
		return s.buildDocumentSymbols()
	}
	s.queries.documentSymbolsOnce.Do(func() {
		s.queries.documentSymbols = s.buildDocumentSymbols()
	})
	return *s.queries.documentSymbols.Clone()
}

func (s Snapshot) cachedReferences(target referenceTarget, includeDeclaration bool) (list.List[Location], bool) {
	if target.id == "" {
		return list.List[Location]{}, false
	}
	if s.queries == nil {
		return s.buildReferences(target, includeDeclaration)
	}
	key := referenceTargetKey(target)
	if cached, ok := s.lookupCachedReferences(key, includeDeclaration); ok {
		return cached, cached.Len() > 0
	}
	uses, withDecl := s.buildReferenceLists(target)
	s.storeCachedReferences(key, uses, withDecl)
	if includeDeclaration {
		return withDecl, withDecl.Len() > 0
	}
	return uses, uses.Len() > 0
}

func (s Snapshot) buildDocumentSymbols() list.List[DocumentSymbol] {
	if s.Result.Binding == nil {
		return list.List[DocumentSymbol]{}
	}
	return documentSymbolsFromEntries(s.topLevelDocumentSymbolEntries())
}

func (s Snapshot) buildReferences(target referenceTarget, includeDeclaration bool) (list.List[Location], bool) {
	uses, withDecl := s.buildReferenceLists(target)
	if includeDeclaration {
		return withDecl, withDecl.Len() > 0
	}
	return uses, uses.Len() > 0
}

func (s Snapshot) buildReferenceLists(target referenceTarget) (list.List[Location], list.List[Location]) {
	uses := list.NewList[Location]()
	withDecl := list.NewList[Location]()
	if location, ok := s.referenceDeclarationLocation(target); ok {
		withDecl.Add(location)
	}
	if s.Result.Binding == nil || s.Result.Binding.Uses == nil {
		return *uses, *withDecl
	}
	s.Result.Binding.Uses.Range(func(_ string, use compiler.NameUse) bool {
		if use.Kind != target.kind || use.TargetID != target.id {
			return true
		}
		location, ok := s.locationForSpan(use.Pos, use.End)
		if ok {
			uses.Add(location)
			withDecl.Add(location)
		}
		return true
	})
	return *uses, *withDecl
}

func (s Snapshot) lookupCachedReferences(key string, includeDeclaration bool) (list.List[Location], bool) {
	if s.queries == nil {
		return list.List[Location]{}, false
	}
	s.queries.referencesMu.RLock()
	defer s.queries.referencesMu.RUnlock()
	if s.queries.referenceComputed == nil || !s.queries.referenceComputed.Contains(key) {
		return list.List[Location]{}, false
	}
	if includeDeclaration {
		items := s.queries.referenceWithDecl.GetCopy(key)
		return *list.NewList(items...), true
	}
	items := s.queries.referenceUses.GetCopy(key)
	return *list.NewList(items...), true
}

func (s Snapshot) storeCachedReferences(key string, uses, withDecl list.List[Location]) {
	if s.queries == nil {
		return
	}
	s.queries.referencesMu.Lock()
	defer s.queries.referencesMu.Unlock()
	if s.queries.referenceComputed != nil && s.queries.referenceComputed.Contains(key) {
		return
	}
	if s.queries.referenceUses == nil {
		s.queries.referenceUses = mapping.NewMultiMap[string, Location]()
	}
	if s.queries.referenceWithDecl == nil {
		s.queries.referenceWithDecl = mapping.NewMultiMap[string, Location]()
	}
	if s.queries.referenceComputed == nil {
		s.queries.referenceComputed = set.NewSet[string]()
	}
	if uses.Len() > 0 {
		s.queries.referenceUses.PutAll(key, uses.Values()...)
	}
	if withDecl.Len() > 0 {
		s.queries.referenceWithDecl.PutAll(key, withDecl.Values()...)
	}
	s.queries.referenceComputed.Add(key)
}

func referenceTargetKey(target referenceTarget) string {
	return string(target.kind) + "\x00" + target.id
}
