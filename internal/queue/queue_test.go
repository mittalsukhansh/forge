package queue

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mittalsukhansh/forge/internal/pipeline"
	"github.com/mittalsukhansh/forge/internal/store"
)

func newTestQueue(t *testing.T, workers int) (*Queue, *store.Store) {
	t.Helper()

	s, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	q := New(ctx, s, workers)

	t.Cleanup(func() {
		cancel()
		q.Wait()
		s.Close()
	})

	return q, s
}

func TestQueueCapacity(t *testing.T) {
	q, _ := newTestQueue(t, 0)

	p := &pipeline.Pipeline{Name: "Capacity Test"}

	for i := 0; i < 32; i++ {
		if _, err := q.Submit(p); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := q.Submit(p); err != ErrQueueFull {
		t.Fatal("Expected queue capacity error")
	}
}

func TestQueueExecution(t *testing.T) {
	if os.Getenv("FORGE_DOCKER_TESTS") != "1" {
		t.Skip("Docker integration tests disabled")
	}

	q, s := newTestQueue(t, 2)

	p := &pipeline.Pipeline{
		Name: "Worker Test",
		Steps: []pipeline.Step{
			{
				Name:  "Build",
				Image: "alpine:3",
				Commands: []string{
					"echo worker-success",
				},
			},
		},
	}

	id, err := q.Submit(p)
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		job, err := s.Get(id)
		if err != nil {
			t.Fatal(err)
		}

		if job.Status == "succeeded" {
			if job.Logs == "" {
				t.Fatal("Execution logs missing")
			}
			return
		}

		if job.Status == "failed" {
			t.Fatal(job.Error)
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatal("Job did not complete before deadline")
}
