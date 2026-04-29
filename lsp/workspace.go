package lsp

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sync"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
)

type Workspace struct {
	base *compiler.Compiler

	mu   sync.RWMutex
	docs *mapping.OrderedMap[string, Document]
}

func NewWorkspace(opts Options) *Workspace {
	base := opts.baseCompiler()
	return &Workspace{
		base: base.Clone(),
		docs: mapping.NewOrderedMap[string, Document](),
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
	sources := mapping.NewMapWithCapacity[string, []byte](docs.Len())
	readFile := func(name string) ([]byte, error) {
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

	analysisCompiler := w.base.Clone()
	analysisCompiler.SetReadFile(readFile)
	result := analysisCompiler.CompileFileDetailed(ctx, path)
	var version int32
	if doc, ok := docs.Get(path); ok {
		version = doc.Version
		uri = doc.URI
	}
	if uri == "" {
		uri = FileURI(path)
	}
	if _, ok := sources.Get(path); !ok {
		if doc, ok := docs.Get(path); ok {
			sources.Set(path, slices.Clone(doc.Text))
		}
	}
	files, fileSpans := buildFileSetIndex(result.FileSet)
	return Snapshot{
		URI:         uri,
		Path:        path,
		Version:     version,
		Result:      result,
		Diagnostics: diagnosticsFromResult(result, sources),
		compiler:    analysisCompiler,
		documents:   documentsByPath(docs),
		files:       files,
		fileSpans:   fileSpans,
		sources:     sources,
	}, nil
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
