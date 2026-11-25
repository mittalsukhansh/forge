package store

import (
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()

	s, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		s.Close()
	})

	return s
}

func TestJobLifecycle(t *testing.T) {
	s := testStore(t)

	id, err := s.Create("Test Job")
	if err != nil {
		t.Fatal(err)
	}

	job, err := s.Get(id)
	if err != nil || job.Status != "queued" {
		t.Fatal("Expected queued job", err)
	}

	if err := s.Start(id); err != nil {
		t.Fatal(err)
	}

	job, err = s.Get(id)
	if err != nil || job.Status != "running" {
		t.Fatal("Expected running job", err)
	}

	if err := s.AppendLogs(id, "Build output"); err != nil {
		t.Fatal(err)
	}

	if err := s.Finish(id, "succeeded", ""); err != nil {
		t.Fatal(err)
	}

	job, err = s.Get(id)

	if err != nil {
		t.Fatal(err)
	}

	if job.Status != "succeeded" {
		t.Fatal("Incorrect final status")
	}

	if job.Logs != "Build output" {
		t.Fatal("Logs were not persisted")
	}
}

func TestFailedJob(t *testing.T) {
	s := testStore(t)

	id, err := s.Create("Failed Build")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Finish(id, "failed", "exit status 1"); err != nil {
		t.Fatal(err)
	}

	job, err := s.Get(id)

	if err != nil {
		t.Fatal(err)
	}

	if job.Status != "failed" || job.Error != "exit status 1" {
		t.Fatal("Failure information not recorded")
	}
}

func TestListJobs(t *testing.T) {
	s := testStore(t)

	for i := 0; i < 3; i++ {
		if _, err := s.Create("Test Build"); err != nil {
			t.Fatal(err)
		}
	}

	jobs, err := s.List()

	if err != nil {
		t.Fatal(err)
	}

	if len(jobs) != 3 {
		t.Fatal("Incorrect job count")
	}

	if jobs[0].ID <= jobs[1].ID {
		t.Fatal("Incorrect job ordering")
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.db")

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	id, err := s.Create("Persistent Build")
	if err != nil {
		s.Close()
		t.Fatal(err)
	}

	if err := s.Finish(id, "succeeded", ""); err != nil {
		s.Close()
		t.Fatal(err)
	}

	s.Close()

	reopened, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	job, err := reopened.Get(id)

	if err != nil || job.Status != "succeeded" {
		t.Fatal("Job did not survive restart", err)
	}
}

func TestRestartRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.db")

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	id, err := s.Create("Interrupted Build")
	if err != nil {
		s.Close()
		t.Fatal(err)
	}

	if err := s.Start(id); err != nil {
		s.Close()
		t.Fatal(err)
	}

	s.Close()

	reopened, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	job, err := reopened.Get(id)

	if err != nil || job.Status != "failed" {
		t.Fatal("Interrupted build was not recovered", err)
	}
}
