package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

func resolveImportPaths(from, spec string) ([]string, error) {
	baseDir := filepath.Dir(from)
	if !containsGlobMeta(spec) {
		return []string{filepath.Clean(filepath.Join(baseDir, spec))}, nil
	}

	pattern := filepath.ToSlash(filepath.Join(baseDir, spec))
	base, globPattern := doublestar.SplitPattern(pattern)
	matches, err := doublestar.Glob(os.DirFS(base), globPattern)
	if err != nil {
		return nil, oops.Wrapf(err, "resolve import pattern %q", spec)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("import pattern %q matched no files", spec)
	}

	paths := lo.Map(matches, func(match string, _ int) string {
		return filepath.Clean(filepath.Join(base, filepath.FromSlash(match)))
	})

	files := make([]string, 0, len(paths))
	for _, candidate := range paths {
		info, err := os.Stat(candidate)
		if err != nil {
			return nil, oops.Wrapf(err, "stat import candidate %q", candidate)
		}
		if info.IsDir() {
			continue
		}
		files = append(files, candidate)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("import pattern %q matched no files", spec)
	}
	return files, nil
}

func containsGlobMeta(spec string) bool {
	return strings.ContainsAny(spec, "*?[]{}")
}
