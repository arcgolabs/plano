package servicedsl_test

import (
	"context"
	"os"
	"testing"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/examples/servicedsl"
)

func TestLowerSample(t *testing.T) {
	src := mustReadServiceSample(t)
	stack := compileStack(t, src)
	assertStackName(t, stack, "demo")
	assertServiceOrder(t, stack, []string{"db", "api", "worker"})
	service := requireService(t, stack, "api")
	assertDependsOn(t, service, []string{"db"})
	assertEnvValue(t, service, "LOG_LEVEL", "info")
}

func mustReadServiceSample(t *testing.T) []byte {
	t.Helper()
	src, err := os.ReadFile("sample.plano")
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func compileStack(t *testing.T, src []byte) *servicedsl.Stack {
	t.Helper()
	c := compiler.New(compiler.Options{
		LookupEnv: func(key string) (string, bool) {
			if key == "LOG_LEVEL" {
				return "info", true
			}
			return "", false
		},
	})
	if err := servicedsl.Register(c); err != nil {
		t.Fatal(err)
	}
	result := c.CompileSourceDetailed(context.Background(), "sample.plano", src)
	if result.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	if result.HIR == nil {
		t.Fatal("expected HIR")
	}
	stack, err := servicedsl.Lower(result.HIR)
	if err != nil {
		t.Fatal(err)
	}
	return stack
}

func assertStackName(t *testing.T, stack *servicedsl.Stack, want string) {
	t.Helper()
	if stack.Name != want {
		t.Fatalf("name = %q", stack.Name)
	}
}

func assertServiceOrder(t *testing.T, stack *servicedsl.Stack, want []string) {
	t.Helper()
	got := stack.Services.Keys()
	if len(got) != len(want) {
		t.Fatalf("service order = %#v, want %#v", got, want)
	}
	for idx, item := range want {
		if got[idx] != item {
			t.Fatalf("service order = %#v, want %#v", got, want)
		}
	}
}

func requireService(t *testing.T, stack *servicedsl.Stack, name string) servicedsl.Service {
	t.Helper()
	service, ok := stack.Services.Get(name)
	if !ok {
		t.Fatalf("expected %s service", name)
	}
	return service
}

func assertDependsOn(t *testing.T, service servicedsl.Service, want []string) {
	t.Helper()
	if len(service.DependsOn) != len(want) {
		t.Fatalf("depends_on = %#v, want %#v", service.DependsOn, want)
	}
	for idx, item := range want {
		if service.DependsOn[idx] != item {
			t.Fatalf("depends_on = %#v, want %#v", service.DependsOn, want)
		}
	}
}

func assertEnvValue(t *testing.T, service servicedsl.Service, key, want string) {
	t.Helper()
	value, ok := service.Env.Get(key)
	if !ok || value != want {
		t.Fatalf("env %s = %#v", key, value)
	}
}
