package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Martins6/textclaw/internal/config"
	"github.com/Martins6/textclaw/internal/daemon/commands"
	"github.com/Martins6/textclaw/internal/daemon/heartbeat"
	"github.com/Martins6/textclaw/internal/daemon/listener"
	"github.com/Martins6/textclaw/internal/daemon/logs"
	"github.com/Martins6/textclaw/internal/daemon/provisioner"
	"github.com/Martins6/textclaw/internal/daemon/router"
	"github.com/Martins6/textclaw/internal/daemon/runner"
	"github.com/Martins6/textclaw/internal/database"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	gray   = "\033[90m"
)

func channelLog(workspaceID, prefix, color, msg string) {
	timestamp := time.Now().Format("15:04:05")
	consoleMsg := fmt.Sprintf("%s[%s]%s %s%s%s %s\n", gray, timestamp, reset, color, prefix, reset, msg)
	fmt.Fprint(os.Stderr, consoleMsg)
	logs.Log(workspaceID, prefix, msg)
}

func channelIn(workspaceID, chatID, sender, content string) {
	preview := content
	if len(preview) > 80 {
		preview = preview[:77] + "..."
	}
	channelLog(workspaceID, "INPUT", green, fmt.Sprintf("[%s] %s: %s", chatID, sender, preview))
}

func channelOut(workspaceID, chatID, content string) {
	preview := content
	if len(preview) > 80 {
		preview = preview[:77] + "..."
	}
	channelLog(workspaceID, "OUTPUT", cyan, fmt.Sprintf("[%s] %s", chatID, preview))
}

func daemonLog(workspaceID, msg string) {
	timestamp := time.Now().Format("15:04:05")
	consoleMsg := fmt.Sprintf("%s[%s]%s %sDAEMON%s %s\n", gray, timestamp, reset, yellow, reset, msg)
	fmt.Fprint(os.Stderr, consoleMsg)
	logs.Log(workspaceID, "DAEMON", msg)
}

type Daemon struct {
	cfg                *config.Config
	db                 *database.DB
	adapter            listener.Adapter
	router             *router.Router
	provisioner        *provisioner.Provisioner
	runner             *runner.Runner
	workspaceBase      string
	heartbeatScheduler *heartbeat.Scheduler
	commandHandler     *commands.Handler
}

func New(cfgPath string) (*Daemon, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	workspaceBase := cfg.Workspace.BasePath
	if workspaceBase == "" {
		workspaceBase = filepath.Join(homeDir, ".textclaw", "workspaces")
	}

	dbPath := filepath.Join(homeDir, ".textclaw", "database", "sqlite.db")
	db, err := database.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := database.InitSchema(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	r := router.New(db)

	openCodeConfigDir := filepath.Join(homeDir, ".textclaw", "opencode-config")
	openCodeAuthDir := filepath.Join(homeDir, ".textclaw", "opencode-auth")
	opencodeDotPath := filepath.Join(homeDir, ".textclaw", ".opencode")

	mainUserID := ""
	if cfg.Main.Enabled {
		mainUserID = cfg.Main.TelegramID

		filesDir := filepath.Join(workspaceBase, "files")
		if err := os.MkdirAll(filesDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create main user files directory: %w", err)
		}

		stateDir := filepath.Join(workspaceBase, "opencode-state")
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create main user state directory: %w", err)
		}
	}

	p := provisioner.New(db, workspaceBase, "", opencodeDotPath, mainUserID)

	agent := cfg.Main.Agent
	provider := cfg.Main.Provider
	model := cfg.Main.Model

	runr, err := runner.New(workspaceBase, filepath.Join(homeDir, ".textclaw"), openCodeConfigDir, openCodeAuthDir, db, mainUserID, agent, provider, model, runner.WithImage(cfg.Container.Image))
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	hbScheduler := heartbeat.NewScheduler(runr, db, cfg, workspaceBase)

	d := &Daemon{
		cfg:                cfg,
		db:                 db,
		router:             r,
		provisioner:        p,
		runner:             runr,
		workspaceBase:      workspaceBase,
		heartbeatScheduler: hbScheduler,
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

	daemonLog("main", fmt.Sprintf("Starting TextClaw daemon with %s adapter", adapter.Name()))

	cmdRegistry := commands.NewRegistry(d.db)
	if err := cmdRegistry.SeedDefaultCommands(); err != nil {
		daemonLog("main", fmt.Sprintf("Failed to seed default commands: %v", err))
	}
	d.commandHandler = commands.NewHandler(cmdRegistry, d.runner, adapter)

	if err := d.heartbeatScheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start heartbeat scheduler: %w", err)
	}

	if err := d.loadHeartbeatWorkspaces(); err != nil {
		daemonLog("main", fmt.Sprintf("Failed to load heartbeat workspaces: %v", err))
	}

	daemonLog("main", "Starting all workspace containers...")
	if err := d.runner.StartAllContainers(ctx); err != nil {
		daemonLog("main", fmt.Sprintf("Failed to start workspace containers: %v", err))
	}

	go func() {
		if err := adapter.Listen(ctx, d.handleMessage); err != nil && err != context.Canceled {
			daemonLog("main", fmt.Sprintf("Listener error: %v", err))
		}
	}()

	return nil
}

func (d *Daemon) loadHeartbeatWorkspaces() error {
	workspaces, err := database.GetAllWorkspaces(d.db)
	if err != nil {
		return err
	}

	for _, ws := range workspaces {
		wsCfg, err := config.LoadWorkspaceConfig(filepath.Join(d.workspaceBase, ws.ID))
		if err != nil {
			daemonLog("main", fmt.Sprintf("Failed to load config for workspace %s: %v", ws.ID, err))
			continue
		}
		if wsCfg != nil && wsCfg.Heartbeat != nil && wsCfg.Heartbeat.Enabled {
			if err := d.heartbeatScheduler.AddWorkspace(ws.ID, wsCfg.Heartbeat.Schedule); err != nil {
				daemonLog("main", fmt.Sprintf("Failed to add heartbeat for workspace %s: %v", ws.ID, err))
			}
		}
	}
	return nil
}

func (d *Daemon) handleMessage(ctx context.Context, msg listener.Message) error {
	workspaceID, err := d.router.Lookup(msg.Sender)
	if err != nil {
		if err == router.ErrContactNotFound {
			daemonLog("main", fmt.Sprintf("New contact %s, provisioning workspace", msg.Sender))
			workspaceID, err = d.provisioner.EnsureWorkspace(msg.Sender)
			if err != nil {
				daemonLog("main", fmt.Sprintf("Failed to provision workspace: %v", err))
				return fmt.Errorf("failed to provision workspace: %w", err)
			}
			daemonLog("main", fmt.Sprintf("Created workspace %s for contact %s", workspaceID, msg.Sender))
		} else {
			daemonLog("main", fmt.Sprintf("Failed to lookup contact: %v", err))
			return fmt.Errorf("failed to lookup contact: %w", err)
		}
	}

	channelIn(workspaceID, msg.ChatID, msg.Sender, msg.Content)

	daemonLog(workspaceID, fmt.Sprintf("Routing message to workspace %s", workspaceID))

	if err := database.SaveMessage(d.db, &database.Message{
		WorkspaceID: workspaceID,
		ContactID:   msg.Sender,
		Content:     msg.Content,
		ContentType: msg.ContentType,
		Direction:   "incoming",
	}); err != nil {
		daemonLog(workspaceID, fmt.Sprintf("Failed to save message: %v", err))
	}

	if d.commandHandler != nil {
		handled, err := d.commandHandler.HandleCommand(ctx, msg, workspaceID)
		if err != nil {
			daemonLog(workspaceID, fmt.Sprintf("Command handler error: %v", err))
		}
		if handled {
			return nil
		}
	}

	daemonLog(workspaceID, fmt.Sprintf("Executing prompt in workspace %s", workspaceID))
	result, err := d.runner.Execute(ctx, workspaceID, msg.Content)
	if err != nil {
		errMsg := fmt.Sprintf("Error: %v", err)
		daemonLog(workspaceID, fmt.Sprintf("Failed to execute: %v", err))
		return d.adapter.Send(msg.ChatID, errMsg)
	}

	response := result.Response
	if result.SessionRecreated {
		response = "Session expired. Started a new session - previous context cleared.\n\n" + response
	}

	channelOut(workspaceID, msg.ChatID, response)

	if err := database.SaveMessage(d.db, &database.Message{
		WorkspaceID: workspaceID,
		ContactID:   msg.Sender,
		Content:     response,
		ContentType: "text",
		Direction:   "outgoing",
	}); err != nil {
		daemonLog(workspaceID, fmt.Sprintf("Failed to save response: %v", err))
	}

	return d.adapter.Send(msg.ChatID, response)
}

func (d *Daemon) Stop() error {
	if d.heartbeatScheduler != nil {
		if err := d.heartbeatScheduler.Stop(); err != nil {
			daemonLog("main", fmt.Sprintf("Failed to stop heartbeat scheduler: %v", err))
		}
	}
	if d.runner != nil {
		d.runner.Close()
	}
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
	daemonLog("main", "Shutting down...")

	cancel()

	return d.Stop()
}
