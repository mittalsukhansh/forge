package engine

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mittalsukhansh/forge/internal/pipeline"
)

func requireDocker(t *testing.T) {
	t.Helper()

	if os.Getenv("FORGE_DOCKER_TESTS") != "1" {
		t.Skip("Docker integration tests disabled")
	}
}

func TestContainerName(t *testing.T) {
	first, err := containerName()
	if err != nil {
		t.Fatal(err)
	}

	second, err := containerName()
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Fatal("Container names must be unique")
	}

	if !strings.HasPrefix(first, "forge-") {
		t.Fatal("Invalid container name")
	}
}

func TestSuccessfulPipeline(t *testing.T) {
	requireDocker(t)

	p := &pipeline.Pipeline{
		Name: "Integration Test",
		Steps: []pipeline.Step{
			{
				Name:     "Build",
				Image:    "alpine:3",
				Commands: []string{"echo build-success"},
			},
			{
				Name:     "Test",
				Image:    "alpine:3",
				Commands: []string{"echo test-success"},
			},
		},
	}

	var output bytes.Buffer

	err := New().Run(context.Background(), p, &output)
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{
		"build-success",
		"test-success",
		"[PIPELINE SUCCESS]",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("Missing output: %s", expected)
		}
	}
}

func TestFailedPipeline(t *testing.T) {
	requireDocker(t)

	p := &pipeline.Pipeline{
		Name: "Failure Test",
		Steps: []pipeline.Step{
			{
				Name:     "Fail",
				Image:    "alpine:3",
				Commands: []string{"exit 1"},
			},
			{
				Name:     "Publish",
				Image:    "alpine:3",
				Commands: []string{"echo should-not-run"},
			},
		},
	}

	var output bytes.Buffer

	err := New().Run(context.Background(), p, &output)

	if err == nil {
		t.Fatal("Expected pipeline failure")
	}

	if strings.Contains(output.String(), "should-not-run") {
		t.Fatal("Pipeline continued after failure")
	}

	if !strings.Contains(output.String(), "[PIPELINE FAILED]") {
		t.Fatal("Missing failure log")
	}
}

func TestStepTimeout(t *testing.T) {
	requireDocker(t)

	e := New()
	e.StepTimeout = 2 * time.Second

	step := pipeline.Step{
		Name:     "Timeout Test",
		Image:    "alpine:3",
		Commands: []string{"sleep 30"},
	}

	var output bytes.Buffer

	err := e.RunStep(context.Background(), step, &output)

	if err == nil {
		t.Fatal("Expected timeout")
	}

	if !strings.Contains(err.Error(), "timed out") {
		t.Fatal("Unexpected error:", err)
	}
}
