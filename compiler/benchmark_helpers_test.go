package compiler_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func benchmarkFiles(tb testing.TB) (string, string) {
	tb.Helper()
	dir := tb.TempDir()
	imported := filepath.Join(dir, "tasks.plano")
	root := filepath.Join(dir, "build.plano")
	writeBenchmarkFile(tb, imported, `
task prepare {}
`)
	writeBenchmarkFile(tb, root, `
import "tasks.plano"

workspace {
  name = "bench"
  default = build
}

task build {
  deps = [prepare]

  run {
    exec("go", "test", "./...")
  }
}
`)
	return root, imported
}

func benchmarkGlobFiles(tb testing.TB, count int) string {
	tb.Helper()
	dir := tb.TempDir()
	root := filepath.Join(dir, "build.plano")
	for index := range count {
		subdir := filepath.Join(dir, "tasks", "group")
		file := filepath.Join(subdir, taskFileName(index))
		writeBenchmarkFile(tb, file, taskFileSource(index))
	}
	writeBenchmarkFile(tb, root, benchmarkGlobRootSource())
	return root
}

func benchmarkDiagnosticSource() string {
	return `
fn output(name: string): path {
  return 1
}

task build {
  let enabled = true
  enabled = 1
  outputs = [output("demo")]

  if 1 {
    outputs = ["dist/demo"]
  }

  run {
    exec(1)
  }
}
`
}

func benchmarkGlobRootSource() string {
	return `
import "tasks/**/*.plano"

workspace {
  name = "bench"
  default = task_00
}
`
}

func benchmarkDeepImportGraphFiles(tb testing.TB, depth, fanout int) string {
	tb.Helper()
	dir := tb.TempDir()
	root := filepath.Join(dir, "build.plano")
	entry := filepath.Join(dir, "graph", "layer_00", "node_00.plano")
	writeImportGraphNode(tb, dir, entry, 0, depth, fanout, 0)
	writeBenchmarkFile(tb, root, `
import "graph/layer_00/node_00.plano"

workspace {
  name = "bench"
  default = layer_00_node_00
}
`)
	return root
}

func writeImportGraphNode(
	tb testing.TB,
	rootDir string,
	path string,
	layer int,
	maxDepth int,
	fanout int,
	index int,
) {
	tb.Helper()
	symbol := fmt.Sprintf("layer_%02d_node_%02d", layer, index)
	var builder strings.Builder
	children := make([]string, 0, fanout)
	if layer+1 < maxDepth {
		for childIndex := range fanout {
			globalIndex := index*fanout + childIndex
			childPath := filepath.Join(
				rootDir,
				"graph",
				fmt.Sprintf("layer_%02d", layer+1),
				fmt.Sprintf("node_%02d.plano", globalIndex),
			)
			writeImportGraphNode(tb, rootDir, childPath, layer+1, maxDepth, fanout, globalIndex)
			rel, err := filepath.Rel(filepath.Dir(path), childPath)
			if err != nil {
				tb.Fatal(err)
			}
			mustFprintf(&builder, "import %q\n", filepath.ToSlash(rel))
			children = append(children, fmt.Sprintf("layer_%02d_node_%02d", layer+1, globalIndex))
		}
		mustWriteString(&builder, "\n")
	}
	mustFprintf(&builder, "task %s {\n", symbol)
	if len(children) == 0 {
		mustWriteString(&builder, "  deps = []\n")
	} else {
		mustFprintf(&builder, "  deps = [%s]\n", strings.Join(children, ", "))
	}
	mustFprintf(&builder, "  outputs = [join_path(\"dist\", %q)]\n\n", symbol)
	mustWriteString(&builder, "  run {\n")
	mustFprintf(&builder, "    exec(\"echo\", %q)\n", symbol)
	mustWriteString(&builder, "  }\n}\n")
	writeBenchmarkFile(tb, path, builder.String())
}

func benchmarkControlFlowSource(taskCount int) string {
	var builder strings.Builder
	mustWriteString(&builder, `
fn packages(): list<string> {
  let base = append(["./..."], "./cmd/...")
  if has(merge({unit = "./..."}, {cli = "./cmd/..."}), "cli") {
    base = concat(base, ["./internal/..."])
  }
  return base
}

workspace {
  name = "bench"
  default = build_00
}
`)
	for index := range taskCount {
		deps := "[]"
		if index > 0 {
			deps = fmt.Sprintf("[build_%02d]", index-1)
		}
		mustFprintf(&builder, `
task build_%02d {
  deps = %s
  let outputs_map = merge(
    {primary = join_path("dist", "artifact_%02d")},
    {backup = join_path("dist", "artifact_%02d_backup")},
  )
  outputs = values(outputs_map)

  for idx, pkg in packages() {
    if idx == 1 {
      continue
    }
    if has(["./internal/..."], pkg) {
      break
    }
    run {
      exec("go", "test", pkg)
    }
  }

  for name, output in outputs_map {
    if name == "backup" {
      break
    }
    run {
      exec("echo", output)
    }
  }
}
`, index, deps, index, index)
	}
	return builder.String()
}

func taskFileName(index int) string {
	return "task_" + leftPad2(index) + ".plano"
}

func taskFileSource(index int) string {
	name := "task_" + leftPad2(index)
	return `
task ` + name + ` {
  outputs = [join_path("dist", "` + name + `")]

  run {
    exec("go", "test", "./...")
  }
}
`
}

func leftPad2(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func writeBenchmarkFile(tb testing.TB, path, contents string) {
	tb.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		tb.Fatal(err)
	}
}

func mustWriteString(builder *strings.Builder, value string) {
	if _, err := builder.WriteString(value); err != nil {
		panic(err)
	}
}

func mustFprintf(builder *strings.Builder, format string, args ...any) {
	if _, err := fmt.Fprintf(builder, format, args...); err != nil {
		panic(err)
	}
}
