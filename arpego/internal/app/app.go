package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/insights"
	ilog "github.com/alexey-pronkin/symphony-go/arpego/internal/logging"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/orchestrator"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/securityscan"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/server"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/workflow"
)

// Run starts the Arpego service entrypoint.
func Run() error {
	return RunArgs(os.Args[1:])
}

func RunArgs(args []string) error {
	workflowPath, cliPort, cliPortSet, err := parseArgs(args)
	if err != nil {
		return err
	}

	def, err := workflow.Load(workflowPath)
	if err != nil {
		return err
	}
	cfg := config.New(def.Config)
	if err := config.ValidateDispatch(cfg); err != nil {
		return err
	}

	logger := ilog.Default("")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	trackerClient, taskPlatform, trackerCloser, err := buildTrackerServices(ctx, cfg, workflowPath)
	if err != nil {
		return err
	}
	defer closeQuietly(trackerCloser)
	observability, err := buildObservabilityServices(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeQuietly(observability)
	orc := orchestrator.New(orchestrator.Options{
		Config:   cfg,
		Workflow: def,
		Logger:   logger,
		Tracker:  trackerClient,
		Events:   observability.runtimeEvents,
	})
	if err := orc.Start(ctx); err != nil {
		return err
	}
	defer orc.Stop()

	watchCloser, err := workflow.Watch(workflowPath, def, func(updated *workflow.Definition) {
		orc.ApplyWorkflow(updated)
	})
	if err != nil {
		logger.Warn("workflow_watch outcome=disabled", "path", workflowPath, "reason", err)
	}
	if watchCloser != nil {
		defer closeQuietly(watchCloser)
	}

	var httpServer *server.Server
	if port, ok := resolvePort(cliPort, cliPortSet, def.Config); ok {
		var workspaceScanner server.WorkspaceSecurityScanner
		if cfg.SecurityWorkspaceScanEnabled() {
			workspaceScanner = securityscan.NewTrivyScanner(
				cfg.SecurityWorkspaceScanCommand(),
				time.Duration(cfg.SecurityWorkspaceScanTimeoutMs())*time.Millisecond,
				time.Duration(cfg.SecurityWorkspaceScanTTLSeconds())*time.Second,
			)
		}
		httpServer = server.New(
			orc,
			taskPlatform,
			buildDeliveryInsights(cfg, taskPlatform, orc, observability.deliveryTrends),
			observability.observability,
			workspaceScanner,
			port,
			detectDashboardDir(),
		)
		if err := httpServer.Start(); err != nil {
			return fmt.Errorf("start http server: %w", err)
		}
		logger.Info("http outcome=started", "addr", httpServer.Addr())
	}

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if httpServer != nil {
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Warn("http outcome=shutdown_failed", "reason", err)
		}
	}
	return nil
}

func buildDeliveryInsights(
	cfg config.Config,
	tasks server.TaskPlatform,
	runtime server.Runtime,
	trends insights.TrendStore,
) *insights.Service {
	sources := make([]insights.SourceConfig, 0, len(cfg.InsightsSCMSources()))
	for _, source := range cfg.InsightsSCMSources() {
		sources = append(sources, insights.SourceConfig{
			Kind:       source.Kind,
			Name:       source.Name,
			RepoPath:   source.RepoPath,
			MainBranch: source.MainBranch,
			APIURL:     source.APIURL,
			Repository: source.Repository,
			ProjectID:  source.ProjectID,
			APIToken:   source.APIToken,
		})
	}
	return insights.NewService(insights.Options{
		Tasks:            tasks,
		Runtime:          runtime,
		Trends:           trends,
		Sources:          sources,
		StaleAfter:       time.Duration(cfg.InsightsStaleBranchHours()) * time.Hour,
		ThroughputWindow: time.Duration(cfg.InsightsThroughputWindowDays()) * 24 * time.Hour,
	})
}

func parseArgs(args []string) (string, int, bool, error) {
	workflowPath := workflow.DefaultWorkflowFile
	port := -1
	portSet := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 >= len(args) {
				return "", 0, false, fmt.Errorf("--port requires a value")
			}
			i++
			_, err := fmt.Sscanf(args[i], "%d", &port)
			if err != nil {
				return "", 0, false, fmt.Errorf("invalid --port value %q", args[i])
			}
			portSet = true
		default:
			if len(args[i]) > 0 && args[i][0] == '-' {
				return "", 0, false, fmt.Errorf("unknown flag %q", args[i])
			}
			if workflowPath != workflow.DefaultWorkflowFile {
				return "", 0, false, fmt.Errorf("expected at most one workflow path")
			}
			workflowPath = args[i]
		}
	}

	return workflowPath, port, portSet, nil
}

func resolvePort(cliPort int, cliPortSet bool, raw map[string]any) (int, bool) {
	if cliPortSet {
		return cliPort, true
	}
	serverMap, _ := raw["server"].(map[string]any)
	if serverMap == nil {
		return 0, false
	}
	if _, ok := serverMap["port"]; !ok {
		return 0, false
	}
	return config.New(raw).ServerPort(), true
}

func closeQuietly(closer io.Closer) {
	if closer != nil {
		_ = closer.Close()
	}
}

func detectDashboardDir() string {
	candidates := []string{
		filepath.Join("libretto", "dist"),
		filepath.Join("..", "libretto", "dist"),
	}
	for _, candidate := range candidates {
		indexPath := filepath.Join(candidate, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			return candidate
		}
	}
	return ""
}

func buildTrackerServices(
	ctx context.Context,
	cfg config.Config,
	workflowPath string,
) (orchestrator.Tracker, server.TaskPlatform, io.Closer, error) {
	if cfg.TrackerKind() != "local" {
		return nil, nil, nil, nil
	}
	if cfg.TrackerStorage() == "postgres" {
		postgres, err := tracker.OpenPostgresPlatform(ctx, cfg.StoragePostgresDSN(), cfg.TrackerProjectSlug())
		if err != nil {
			return nil, nil, nil, fmt.Errorf("open postgres task platform: %w", err)
		}
		service := tracker.NewTaskService(postgres, postgres)
		return service, service, postgres, nil
	}
	local := tracker.NewLocalPlatform(resolveTaskFile(workflowPath, cfg), cfg.TrackerProjectSlug())
	service := tracker.NewTaskService(local, local)
	return service, service, nil, nil
}

func buildObservabilityServices(
	ctx context.Context,
	cfg config.Config,
) (observabilityServices, error) {
	if strings.TrimSpace(cfg.StorageClickHouseDSN()) == "" {
		return observabilityServices{}, nil
	}
	runtimeStore, err := tracker.OpenClickHouseObservability(
		ctx,
		cfg.StorageClickHouseDSN(),
		cfg.TrackerProjectSlug(),
	)
	if err != nil {
		return observabilityServices{}, fmt.Errorf("open clickhouse observability: %w", err)
	}
	trendStore, err := insights.OpenClickHouseTrendStore(
		ctx,
		cfg.StorageClickHouseDSN(),
		cfg.TrackerProjectSlug(),
	)
	if err != nil {
		_ = runtimeStore.Close()
		return observabilityServices{}, fmt.Errorf("open clickhouse delivery trends: %w", err)
	}
	return observabilityServices{
		runtimeEvents:  runtimeStore,
		observability:  runtimeStore,
		deliveryTrends: trendStore,
		closer: multiCloser{
			runtimeStore,
			trendStore,
		},
	}, nil
}

type observabilityServices struct {
	runtimeEvents  tracker.RuntimeEventSink
	observability  server.Observability
	deliveryTrends insights.TrendStore
	closer         io.Closer
}

func (s observabilityServices) Close() error {
	if s.closer == nil {
		return nil
	}
	return s.closer.Close()
}

type multiCloser []io.Closer

func (m multiCloser) Close() error {
	var firstErr error
	for _, closer := range m {
		if closer == nil {
			continue
		}
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func resolveTaskFile(workflowPath string, cfg config.Config) string {
	path := strings.TrimSpace(cfg.TrackerFile())
	if path == "" {
		path = "TASKS.yaml"
	}
	if filepath.IsAbs(path) {
		return path
	}
	workflowAbs, err := filepath.Abs(workflowPath)
	if err != nil {
		return path
	}
	return filepath.Join(filepath.Dir(workflowAbs), path)
}
