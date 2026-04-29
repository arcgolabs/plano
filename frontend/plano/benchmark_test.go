package plano_test

import (
	"fmt"
	"go/token"
	"strings"
	"testing"

	"github.com/arcgolabs/plano/frontend/plano"
)

func BenchmarkParseFile(b *testing.B) {
	src := []byte(benchmarkParseSource(12))

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		file, diags := plano.ParseFile(token.NewFileSet(), "bench.plano", src)
		if diags.HasError() {
			b.Fatalf("unexpected diagnostics: %v", diags)
		}
		if file == nil {
			b.Fatal("expected parsed file")
		}
	}
}

func BenchmarkParseLargeFile(b *testing.B) {
	src := []byte(benchmarkParseSource(64))

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		file, diags := plano.ParseFile(token.NewFileSet(), "large.plano", src)
		if diags.HasError() {
			b.Fatalf("unexpected diagnostics: %v", diags)
		}
		if file == nil {
			b.Fatal("expected parsed file")
		}
	}
}

func BenchmarkParseControlFlowFile(b *testing.B) {
	src := []byte(benchmarkParseControlFlowSource(18))

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		file, diags := plano.ParseFile(token.NewFileSet(), "control_flow.plano", src)
		if diags.HasError() {
			b.Fatalf("unexpected diagnostics: %v", diags)
		}
		if file == nil {
			b.Fatal("expected parsed file")
		}
	}
}

func benchmarkParseSource(tasks int) string {
	var builder strings.Builder
	mustWriteString(&builder, `
const target: string = "dist/demo"

workspace {
  name = "demo"
  default = build_00
}
`)
	for index := range tasks {
		mustFprintf(&builder, `
task build_%02d {
  deps = []
  outputs = [join_path("dist", "artifact_%02d")]

  run {
    exec("go", "test", "./...")
  }
}
`, index, index)
	}
	return builder.String()
}

func benchmarkParseControlFlowSource(tasks int) string {
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
  name = "demo"
  default = build_00
}
`)
	for index := range tasks {
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
