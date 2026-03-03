package heartbeat

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestSchedulerAddWorkspace(t *testing.T) {
	scheduler := &Scheduler{
		cron: cron.New(),
		jobs: make(map[string]cron.EntryID),
	}

	err := scheduler.AddWorkspace("test-workspace", "*/5 * * * *")
	if err != nil {
		t.Fatalf("failed to add workspace: %v", err)
	}

	if len(scheduler.jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(scheduler.jobs))
	}

	if _, exists := scheduler.jobs["test-workspace"]; !exists {
		t.Error("expected job for test-workspace to exist")
	}
}

func TestSchedulerAddDuplicateWorkspace(t *testing.T) {
	scheduler := &Scheduler{
		cron: cron.New(),
		jobs: make(map[string]cron.EntryID),
	}

	err := scheduler.AddWorkspace("test-workspace", "*/5 * * * *")
	if err != nil {
		t.Fatalf("failed to add workspace: %v", err)
	}

	err = scheduler.AddWorkspace("test-workspace", "*/10 * * * *")
	if err != nil {
		t.Fatalf("failed to add duplicate workspace: %v", err)
	}

	if len(scheduler.jobs) != 1 {
		t.Errorf("expected 1 job after duplicate add, got %d", len(scheduler.jobs))
	}
}

func TestSchedulerRemoveWorkspace(t *testing.T) {
	scheduler := &Scheduler{
		cron: cron.New(),
		jobs: make(map[string]cron.EntryID),
	}

	scheduler.AddWorkspace("test-workspace", "*/5 * * * *")

	err := scheduler.RemoveWorkspace("test-workspace")
	if err != nil {
		t.Fatalf("failed to remove workspace: %v", err)
	}

	if len(scheduler.jobs) != 0 {
		t.Errorf("expected 0 jobs after remove, got %d", len(scheduler.jobs))
	}
}

func TestSchedulerRemoveNonExistentWorkspace(t *testing.T) {
	scheduler := &Scheduler{
		cron: cron.New(),
		jobs: make(map[string]cron.EntryID),
	}

	err := scheduler.RemoveWorkspace("non-existent")
	if err != nil {
		t.Fatalf("failed to remove non-existent workspace: %v", err)
	}

	if len(scheduler.jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(scheduler.jobs))
	}
}

func TestSchedulerStartStop(t *testing.T) {
	scheduler := &Scheduler{
		cron: cron.New(),
		jobs: make(map[string]cron.EntryID),
	}

	err := scheduler.Start(nil)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("failed to stop scheduler: %v", err)
	}
}
