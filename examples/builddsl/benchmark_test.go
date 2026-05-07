package builddsl_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/examples/builddsl"
)

func BenchmarkLower(b *testing.B) {
	result := compileBenchmarkProject(b, buildBenchmarkSource())
	benchmarkLowerProject(b, result)
}

func BenchmarkLowerLargeProject(b *testing.B) {
	result := compileBenchmarkProject(b, largeBuildBenchmarkSource(24))
	benchmarkLowerProject(b, result)
}

func benchmarkLowerProject(b *testing.B, result compiler.Result) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		project, err := builddsl.Lower(result.HIR)
		if err != nil {
			b.Fatal(err)
		}
		if project == nil || project.Tasks.Len() == 0 {
			b.Fatal("expected lowered project")
		}
	}
}

func compileBenchmarkProject(tb testing.TB, src string) compiler.Result {
	tb.Helper()
	c := compiler.New(compiler.Options{
		LookupEnv: func(string) (string, bool) { return "", false },
	})
	if err := builddsl.Register(c); err != nil {
		tb.Fatal(err)
	}
	result := c.CompileStringDetailed(context.Background(), "build.plano", src)
	if result.Diagnostics.HasError() {
		tb.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	if result.HIR == nil {
		tb.Fatal("expected HIR")
	}
	return result
}

func buildBenchmarkSource() string {
	return `
fn output(name: string): path {
  return join_path("dist", name)
}

workspace {
  name = "bench"
  default = build
}

task prepare {}

task build {
  deps = [prepare]
  outputs = [output("app")]

  for pkg in ["./...", "./cmd/..."] {
    run {
      exec("go", "test", pkg)
    }
  }
}
`
}

func largeBuildBenchmarkSource(taskCount int) string {
	var builder strings.Builder
	mustWriteBenchmarkString(&builder, `
workspace {
  name = "bench"
  default = prepare_00
}
`)
	for index := range taskCount {
		prevPrepare := ""
		prevBinary := ""
		if index > 0 {
			prevPrepare = fmt.Sprintf(", prepare_%02d", index-1)
			prevBinary = fmt.Sprintf(", build_%02d", index-1)
		}
		mustFprintfBenchmark(&builder, `
task prepare_%02d {
  deps = [%s%s]
  outputs = [join_path("dist", "prepare_%02d")]

  run {
    exec("echo", "prepare_%02d")
  }
}

go.test unit_%02d {
  deps = [prepare_%02d]
  packages = ["./...", "./cmd/..."]
}

go.binary build_%02d {
  deps = [unit_%02d%s]
  main = "./cmd/plano"
  out = join_path("bin", "plano_%02d")
}
`, index, strings.TrimPrefix(prevPrepare, ", "), prevBinary, index, index, index, index, index, index, prevBinary, index)
	}
	return builder.String()
}

func mustWriteBenchmarkString(builder *strings.Builder, value string) {
	if _, err := builder.WriteString(value); err != nil {
		panic(err)
	}
}

func mustFprintfBenchmark(builder *strings.Builder, format string, args ...any) {
	if _, err := fmt.Fprintf(builder, format, args...); err != nil {
		panic(err)
	}
}
