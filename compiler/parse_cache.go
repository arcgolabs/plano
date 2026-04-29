package compiler

import (
	"path/filepath"
	"slices"

	lru "github.com/hashicorp/golang-lru/v2"
)

const defaultParseCacheEntries = 32

type parseCache struct {
	entries *lru.Cache[string, parseCacheEntry]
}

type parseCacheEntry struct {
	input   preparedInput
	digests []sourceDigest
}

func newParseCache(entries int) *parseCache {
	if entries <= 0 {
		return nil
	}
	cache, err := lru.New[string, parseCacheEntry](entries)
	if err != nil {
		return nil
	}
	return &parseCache{entries: cache}
}

func normalizeParseCacheEntries(entries int) int {
	switch {
	case entries < 0:
		return 0
	case entries == 0:
		return defaultParseCacheEntries
	default:
		return entries
	}
}

func (c *parseCache) Clear() {
	if c == nil || c.entries == nil {
		return
	}
	c.entries.Purge()
}

func (c *parseCache) get(key string) (parseCacheEntry, bool) {
	if c == nil || c.entries == nil {
		return parseCacheEntry{}, false
	}
	return c.entries.Get(key)
}

func (c *parseCache) add(key string, input preparedInput, digests []sourceDigest) {
	if c == nil || c.entries == nil {
		return
	}
	c.entries.Add(key, parseCacheEntry{
		input:   clonePreparedInput(input),
		digests: slices.Clone(digests),
	})
}

func fileParseCacheKey(path string) string {
	return "file:" + filepath.Clean(path)
}

func sourceParseCacheKey(filename string) string {
	return "source:" + filepath.Clean(filename)
}

func (c *Compiler) cachedFileInput(path string) (preparedInput, bool) {
	return c.cachedParseInput(fileParseCacheKey(path), nil)
}

func (c *Compiler) storeFileInput(path string, input preparedInput, digests []sourceDigest) {
	c.storeParseInput(fileParseCacheKey(path), input, digests)
}

func (c *Compiler) cachedSourceInput(filename string, src []byte) (preparedInput, bool) {
	return c.cachedParseInput(sourceParseCacheKey(filename), src)
}

func (c *Compiler) storeSourceInput(filename string, input preparedInput, digests []sourceDigest) {
	c.storeParseInput(sourceParseCacheKey(filename), input, digests)
}

func (c *Compiler) cachedParseInput(key string, rootSource []byte) (preparedInput, bool) {
	if c == nil || c.parseCache == nil {
		return preparedInput{}, false
	}
	entry, ok := c.parseCache.get(key)
	if !ok || !c.parseInputValid(entry.digests, rootSource) {
		return preparedInput{}, false
	}
	return clonePreparedInput(entry.input), true
}

func (c *Compiler) storeParseInput(key string, input preparedInput, digests []sourceDigest) {
	if c == nil || c.parseCache == nil {
		return
	}
	c.parseCache.add(key, input, normalizeSourceDigests(digests))
}

func normalizeSourceDigests(digests []sourceDigest) []sourceDigest {
	items := slices.Clone(digests)
	slices.SortFunc(items, func(left, right sourceDigest) int {
		switch {
		case left.Name < right.Name:
			return -1
		case left.Name > right.Name:
			return 1
		default:
			return 0
		}
	})
	return items
}

func (c *Compiler) parseInputValid(digests []sourceDigest, rootSource []byte) bool {
	if len(digests) == 0 {
		return false
	}
	for _, item := range digests {
		if !c.parseDigestValid(item, rootSource) {
			return false
		}
	}
	return true
}

func (c *Compiler) parseDigestValid(item sourceDigest, rootSource []byte) bool {
	if item.Inline {
		return rootSource != nil && digestSource(rootSource) == item.Digest
	}
	src, err := c.ReadFile(item.Name)
	return err == nil && digestSource(src) == item.Digest
}
