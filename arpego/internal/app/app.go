package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/config"
	ilog "github.com/alexey-pronkin/symphony-go/arpego/internal/logging"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/orchestrator"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/server"
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

	orc := orchestrator.New(orchestrator.Options{
		Config:   cfg,
		Workflow: def,
		Logger:   logger,
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
		httpServer = server.New(orc, port)
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
