package heartbeat

import (
	"context"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/textclaw/textclaw/internal/config"
	"github.com/textclaw/textclaw/internal/daemon/runner"
	"github.com/textclaw/textclaw/internal/database"
)

type Scheduler struct {
	cron    *cron.Cron
	jobs    map[string]cron.EntryID
	mu      sync.RWMutex
	runner  *runner.Runner
	db      *database.DB
	cfg     *config.Config
	workDir string
}

func NewScheduler(runner *runner.Runner, db *database.DB, cfg *config.Config, workDir string) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		jobs:    make(map[string]cron.EntryID),
		runner:  runner,
		db:      db,
		cfg:     cfg,
		workDir: workDir,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.cron.Start()
	log.Println("Heartbeat scheduler started")
	return nil
}

func (s *Scheduler) Stop() error {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("Heartbeat scheduler stopped")
	return nil
}

func (s *Scheduler) AddWorkspace(workspaceID, schedule string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[workspaceID]; exists {
		return nil
	}

	_, err := s.cron.AddFunc(schedule, func() {
		s.triggerHeartbeat(context.Background(), workspaceID)
	})
	if err != nil {
		return err
	}

	entryID := s.cron.Entries()[len(s.cron.Entries())-1].ID
	s.jobs[workspaceID] = entryID
	log.Printf("Added heartbeat job for workspace %s with schedule %s", workspaceID, schedule)

	return nil
}

func (s *Scheduler) RemoveWorkspace(workspaceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.jobs[workspaceID]
	if !exists {
		return nil
	}

	s.cron.Remove(entryID)
	delete(s.jobs, workspaceID)
	log.Printf("Removed heartbeat job for workspace %s", workspaceID)

	return nil
}

func (s *Scheduler) TriggerHeartbeat(ctx context.Context, workspaceID string) {
	s.mu.Lock()
	if _, exists := s.jobs[workspaceID]; !exists {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	s.triggerHeartbeat(ctx, workspaceID)
}

func (s *Scheduler) triggerHeartbeat(ctx context.Context, workspaceID string) {
	log.Printf("Triggering heartbeat for workspace %s", workspaceID)

	workspacePath := s.workDir + "/" + workspaceID
	wsCfg, err := config.LoadWorkspaceConfig(workspacePath)
	if err != nil {
		log.Printf("Failed to load workspace config for %s: %v", workspaceID, err)
		return
	}

	if wsCfg == nil || wsCfg.Heartbeat == nil || !wsCfg.Heartbeat.Enabled {
		log.Printf("Heartbeat not enabled for workspace %s", workspaceID)
		return
	}

	prompt := "This is a heartbeat trigger. Please check the HEARTBEATS.md file in your workspace for instructions on what to check. If there's anything worth reporting, respond with the details. If nothing is worth reporting, respond with exactly: No updates"

	response, err := s.runner.Execute(ctx, workspaceID, prompt)
	if err != nil {
		log.Printf("Heartbeat execution failed for workspace %s: %v", workspaceID, err)
		return
	}

	log.Printf("Heartbeat response for workspace %s: %s", workspaceID, response)
}
