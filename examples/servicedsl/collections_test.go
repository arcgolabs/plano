package servicedsl_test

import (
	"os"
	"testing"
)

func TestLowerCollectionsSample(t *testing.T) {
	src, err := os.ReadFile("collections.plano")
	if err != nil {
		t.Fatal(err)
	}

	stack := compileStack(t, src)
	assertStackName(t, stack, "platform")
	assertServiceOrder(t, stack, []string{"db", "api", "worker"})

	api := requireService(t, stack, "api")
	if api.Port != 8080 {
		t.Fatalf("api port = %d", api.Port)
	}
	assertDependsOn(t, api, []string{"db"})
	assertEnvValue(t, api, "OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel:4318")

	worker := requireService(t, stack, "worker")
	assertDependsOn(t, worker, []string{"db", "api"})
	assertEnvValue(t, worker, "QUEUE", "critical")
}
