package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/textclaw/textclaw/internal/config"
	"github.com/textclaw/textclaw/internal/daemon/listener"
	"github.com/textclaw/textclaw/internal/daemon/provisioner"
	"github.com/textclaw/textclaw/internal/daemon/router"
	"github.com/textclaw/textclaw/internal/database"
)

type Daemon struct {
	cfg           *config.Config
	db            *database.DB
	adapter       listener.Adapter
	router        *router.Router
	provisioner   *provisioner.Provisioner
	workspaceBase string
}

func New(cfgPath string) (*Daemon, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	workspaceBase := cfg.Workspace.BasePath
	if workspaceBase == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		workspaceBase = filepath.Join(homeDir, ".textclaw", "workspaces")
	}

	dbPath := filepath.Join(filepath.Dir(cfgPath), "..", "database", "sqlite.db")
	db, err := database.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := database.RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	r := router.New(db)
	p := provisioner.New(db, workspaceBase, "")

	d := &Daemon{
		cfg:           cfg,
		db:            db,
		router:        r,
		provisioner:   p,
		workspaceBase: workspaceBase,
	}

	return d, nil
}

func (d *Daemon) Start(ctx context.Context) error {
	if d.cfg.Telegram.Token == "" {
		return fmt.Errorf("telegram token not configured")
	}

	adapter, err := listener.NewTelegramAdapter(
		d.cfg.Telegram.Token,
		d.workspaceBase,
		d.cfg.Telegram.AllowedUsers,
	)
	if err != nil {
		return fmt.Errorf("failed to create telegram adapter: %w", err)
	}
	d.adapter = adapter

	log.Printf("Starting TextClaw daemon with %s adapter", adapter.Name())

	go func() {
		if err := adapter.Listen(ctx, d.handleMessage); err != nil && err != context.Canceled {
			log.Printf("Listener error: %v", err)
		}
	}()

	return nil
}

func (d *Daemon) handleMessage(ctx context.Context, msg listener.Message) error {
	log.Printf("Received message from %s: %s", msg.Sender, msg.Content)

	workspaceID, err := d.router.Lookup(msg.Sender)
	if err != nil {
		if err == router.ErrContactNotFound {
			log.Printf("New contact %s, provisioning workspace", msg.Sender)
			workspaceID, err = d.provisioner.EnsureWorkspace(msg.Sender)
			if err != nil {
				log.Printf("Failed to provision workspace: %v", err)
				return fmt.Errorf("failed to provision workspace: %w", err)
			}
			log.Printf("Created workspace %s for contact %s", workspaceID, msg.Sender)
		} else {
			log.Printf("Failed to lookup contact: %v", err)
			return fmt.Errorf("failed to lookup contact: %w", err)
		}
	}

	log.Printf("Routing message to workspace %s", workspaceID)

	if err := database.SaveMessage(d.db, &database.Message{
		WorkspaceID: workspaceID,
		ContactID:   msg.Sender,
		Content:     msg.Content,
		ContentType: msg.ContentType,
		Direction:   "incoming",
	}); err != nil {
		log.Printf("Failed to save message: %v", err)
	}

	log.Printf("Message saved to workspace %s (agent runner not implemented yet)", workspaceID)

	return nil
}

func (d *Daemon) Stop() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func Run() error {
	cfgPath := filepath.Join(os.Getenv("HOME"), ".textclaw", "setup.toml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfgPath = "./setup.toml"
	}

	d, err := New(cfgPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := d.Start(ctx); err != nil {
		return err
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down...")

	cancel()

	return d.Stop()
}
