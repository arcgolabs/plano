package lsp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"slices"
	"sync"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
)

type Workspace struct {
	base *compiler.Compiler

	mu        sync.RWMutex
	docs      *mapping.OrderedMap[string, Document]
	snapshots *mapping.Map[string, Snapshot]
}

func NewWorkspace(opts Options) *Workspace {
	base := opts.baseCompiler()
	return &Workspace{
		base:      base.Clone(),
		docs:      mapping.NewOrderedMap[string, Document](),
		snapshots: mapping.NewMap[string, Snapshot](),
	}
}

func (w *Workspace) Open(uri string, version int32, text []byte) error {
	return w.setDocument(uri, version, text)
}

func (w *Workspace) OpenString(uri string, version int32, text string) error {
	return w.Open(uri, version, []byte(text))
}

func (w *Workspace) Update(uri string, version int32, text []byte) error {
	return w.setDocument(uri, version, text)
}

func (w *Workspace) UpdateString(uri string, version int32, text string) error {
	return w.Update(uri, version, []byte(text))
}

func (w *Workspace) Close(uri string) error {
	path, err := PathFromURI(uri)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.docs.Delete(filepath.Clean(path))
	w.snapshots.Clear()
	return nil
}

func (w *Workspace) Document(uri string) (Document, bool) {
	path, err := PathFromURI(uri)
	if err != nil {
		return Document{}, false
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.docs.Get(filepath.Clean(path))
}

func (w *Workspace) Analyze(ctx context.Context, uri string) (Snapshot, error) {
	path, err := PathFromURI(uri)
	if err != nil {
		return Snapshot{}, err
	}
	clean := filepath.Clean(path)
	return w.analyzeDocuments(ctx, uri, clean, w.snapshotDocuments())
}

func (w *Workspace) AnalyzeSource(ctx context.Context, uri string, version int32, text []byte) (Snapshot, error) {
	path, err := PathFromURI(uri)
	if err != nil {
		return Snapshot{}, err
	}
	clean := filepath.Clean(path)
	docs := w.snapshotDocuments()
	docs.Set(clean, Document{
		URI:     uri,
		Path:    clean,
		Version: version,
		Text:    slices.Clone(text),
	})
	return w.analyzeDocuments(ctx, uri, clean, docs)
}

func (w *Workspace) AnalyzeString(ctx context.Context, uri string, version int32, text string) (Snapshot, error) {
	return w.AnalyzeSource(ctx, uri, version, []byte(text))
}

func (w *Workspace) analyzeDocuments(
	ctx context.Context,
	uri string,
	path string,
	docs *mapping.OrderedMap[string, Document],
) (Snapshot, error) {
	cacheKey, cacheable, err := analysisCacheKey(path, docs)
	if err != nil {
		return Snapshot{}, err
	}
	if snapshot, ok := w.cachedAnalysis(cacheable, cacheKey); ok {
		return snapshot, nil
	}

	sources, readFile := w.analysisReader(docs)
	analysisCompiler := w.base.Clone()
	analysisCompiler.SetReadFile(readFile)
	result := analysisCompiler.CompileFileDetailed(ctx, path)
	snapshot := buildSnapshot(path, uri, docs, result, sources, analysisCompiler)
	w.cacheAnalysis(cacheable, cacheKey, snapshot)
	return snapshot, nil
}

func (w *Workspace) setDocument(uri string, version int32, text []byte) error {
	path, err := PathFromURI(uri)
	if err != nil {
		return err
	}
	doc := Document{
		URI:     uri,
		Path:    filepath.Clean(path),
		Version: version,
		Text:    slices.Clone(text),
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.docs.Set(doc.Path, doc)
	w.snapshots.Clear()
	return nil
}

func (w *Workspace) snapshotDocuments() *mapping.OrderedMap[string, Document] {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.docs.Clone()
}

func documentsByPath(items *mapping.OrderedMap[string, Document]) *mapping.Map[string, Document] {
	out := mapping.NewMapWithCapacity[string, Document](items.Len())
	items.Range(func(path string, doc Document) bool {
		out.Set(path, doc)
		return true
	})
	return out
}

func (w *Workspace) cachedAnalysis(cacheable bool, cacheKey string) (Snapshot, bool) {
	if !cacheable {
		return Snapshot{}, false
	}
	return w.cachedSnapshot(cacheKey)
}

func (w *Workspace) cacheAnalysis(cacheable bool, cacheKey string, snapshot Snapshot) {
	if !cacheable {
		return
	}
	w.storeSnapshot(cacheKey, snapshot)
}

func (w *Workspace) analysisReader(
	docs *mapping.OrderedMap[string, Document],
) (*mapping.Map[string, []byte], func(string) ([]byte, error)) {
	sources := mapping.NewMapWithCapacity[string, []byte](docs.Len())
	readFile := func(name string) ([]byte, error) {
		return w.readAnalysisSource(name, docs, sources)
	}
	return sources, readFile
}

func (w *Workspace) readAnalysisSource(
	name string,
	docs *mapping.OrderedMap[string, Document],
	sources *mapping.Map[string, []byte],
) ([]byte, error) {
	current := filepath.Clean(name)
	if doc, ok := docs.Get(current); ok {
		data := slices.Clone(doc.Text)
		sources.Set(current, data)
		return data, nil
	}
	data, err := w.base.ReadFile(current)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", current, err)
	}
	cloned := slices.Clone(data)
	sources.Set(current, cloned)
	return cloned, nil
}

func (w *Workspace) cachedSnapshot(key string) (Snapshot, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.snapshots == nil {
		return Snapshot{}, false
	}
	return w.snapshots.Get(key)
}

func (w *Workspace) storeSnapshot(key string, snapshot Snapshot) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.snapshots == nil {
		w.snapshots = mapping.NewMap[string, Snapshot]()
	}
	w.snapshots.Set(key, snapshot)
}

type analysisDocumentState struct {
	Version int32
	Digest  [32]byte
}

func buildSnapshot(
	path string,
	uri string,
	docs *mapping.OrderedMap[string, Document],
	result compiler.Result,
	sources *mapping.Map[string, []byte],
	analysisCompiler *compiler.Compiler,
) Snapshot {
	version, finalURI := snapshotIdentity(path, uri, docs)
	ensurePrimarySource(path, docs, sources)
	files, fileSpans := buildFileSetIndex(result.FileSet)
	return Snapshot{
		URI:         finalURI,
		Path:        path,
		Version:     version,
		Result:      result,
		Diagnostics: diagnosticsFromResult(result, sources),
		compiler:    analysisCompiler,
		documents:   documentsByPath(docs),
		files:       files,
		fileSpans:   fileSpans,
		sources:     sources,
	}
}

func snapshotIdentity(path, uri string, docs *mapping.OrderedMap[string, Document]) (int32, string) {
	var version int32
	if doc, ok := docs.Get(path); ok {
		version = doc.Version
		uri = doc.URI
	}
	if uri == "" {
		uri = FileURI(path)
	}
	return version, uri
}

func ensurePrimarySource(
	path string,
	docs *mapping.OrderedMap[string, Document],
	sources *mapping.Map[string, []byte],
) {
	if _, ok := sources.Get(path); ok {
		return
	}
	if doc, ok := docs.Get(path); ok {
		sources.Set(path, slices.Clone(doc.Text))
	}
}

func analysisCacheKey(path string, docs *mapping.OrderedMap[string, Document]) (string, bool, error) {
	if docs == nil {
		return "", false, nil
	}
	if _, ok := docs.Get(path); !ok {
		return "", false, nil
	}

	keys := docs.Keys()
	slices.Sort(keys)
	state := mapping.NewOrderedMapWithCapacity[string, analysisDocumentState](len(keys))
	for _, docPath := range keys {
		doc, _ := docs.Get(docPath)
		state.Set(docPath, analysisDocumentState{
			Version: doc.Version,
			Digest:  sha256.Sum256(doc.Text),
		})
	}

	data, err := state.MarshalBinary()
	if err != nil {
		return "", false, fmt.Errorf("marshal workspace analysis cache key: %w", err)
	}
	return path + "\x00" + string(data), true, nil
}
