package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type ClientOption func(*Client)

type Client struct {
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      *bufio.Reader
	stderrDone  chan struct{}
	readTimeout time.Duration
	env         []string
	mu          sync.Mutex
	closed      bool
}

func WithEnv(env []string) ClientOption {
	return func(c *Client) { c.env = env }
}

func WithReadTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) { c.readTimeout = timeout }
}

func NewClient(workspacePath, command string, opts ...ClientOption) (*Client, error) {
	client := &Client{readTimeout: 5 * time.Second, stderrDone: make(chan struct{})}
	for _, opt := range opts {
		opt(client)
	}
	cmd := exec.Command("bash", "-lc", command)
	cmd.Dir = workspacePath
	if len(client.env) > 0 {
		cmd.Env = client.env
	} else {
		cmd.Env = os.Environ()
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	client.cmd = cmd
	client.stdin = stdin
	client.stdout = bufio.NewReaderSize(stdout, 10*1024*1024)
	go func() {
		defer close(client.stderrDone)
		_, _ = io.Copy(io.Discard, stderr)
	}()
	return client, nil
}

func (c *Client) Send(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("client closed")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = c.stdin.Write(append(data, '\n'))
	return err
}

func (c *Client) ReadLine(ctx context.Context, timeout time.Duration) ([]byte, error) {
	if timeout <= 0 {
		timeout = c.readTimeout
	}
	type result struct {
		line []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := c.stdout.ReadBytes('\n')
		ch <- result{line: line, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, &RunError{Kind: ErrResponseTimeout, Message: "timed out waiting for app-server response"}
	case res := <-ch:
		if res.err != nil {
			if errors.Is(res.err, io.EOF) && len(res.line) > 0 {
				return res.line, nil
			}
			return nil, res.err
		}
		return res.line, nil
	}
}

func (c *Client) AwaitResponse(ctx context.Context, id int, timeout time.Duration) (*Response, error) {
	for {
		line, err := c.ReadLine(ctx, timeout)
		if err != nil {
			return nil, err
		}
		var msg Response
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.ID != id {
			continue
		}
		if msg.Error != nil {
			return nil, &RunError{Kind: ErrProtocolPayload, Message: fmt.Sprintf("request %d returned error", id)}
		}
		return &msg, nil
	}
}

func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	_ = c.stdin.Close()
	c.mu.Unlock()

	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	<-c.stderrDone
	_, _ = io.Copy(io.Discard, c.stdout)
	return c.cmd.Wait()
}
