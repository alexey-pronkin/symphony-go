package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

func RunHook(script, cwd string, timeoutMs int) error {
	if script == "" {
		return nil
	}
	if timeoutMs <= 0 {
		timeoutMs = 60_000
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-lc", script)
	cmd.Dir = cwd

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("hook timeout after %dms", timeoutMs)
		}
		if stderr.Len() > 0 {
			return fmt.Errorf("hook failed: %w: %s", err, stderr.String())
		}
		return fmt.Errorf("hook failed: %w", err)
	}

	return nil
}
