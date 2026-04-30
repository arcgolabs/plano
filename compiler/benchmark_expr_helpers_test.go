package compiler_test

import (
	"fmt"
	"strings"
)

func benchmarkExprSource(taskCount int) string {
	var builder strings.Builder
	mustWriteString(&builder, `
const project: string = "demo"

workspace {
  name = project
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
  outputs = [expr("slug(project + '/' + branch)")]

  run {
    exec("echo", "build_%02d")
  }
}
`, index, deps, index)
	}
	return builder.String()
}
