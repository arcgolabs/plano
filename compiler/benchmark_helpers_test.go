package compiler_test

import (
	"os"
	"path/filepath"
	"strconv"
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
