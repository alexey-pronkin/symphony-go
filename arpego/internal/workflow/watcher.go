package workflow

import (
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a WORKFLOW.md file and calls onChange when the file changes.
// Invalid reloads are logged and the last good Definition is preserved.
type Watcher struct {
	path     string
	onChange func(*Definition)
	watcher  *fsnotify.Watcher
	mu       sync.Mutex
	last     *Definition
	done     chan struct{}
}

// Watch starts watching path for changes. onChange is called with the newly
// loaded Definition whenever the file changes successfully.
// The caller must close the returned io.Closer to stop watching.
func Watch(path string, current *Definition, onChange func(*Definition)) (io.Closer, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fw.Add(path); err != nil {
		_ = fw.Close()
		return nil, err
	}

	w := &Watcher{
		path:     path,
		onChange: onChange,
		watcher:  fw,
		last:     current,
		done:     make(chan struct{}),
	}
	go w.loop()
	return w, nil
}

func (w *Watcher) loop() {
	// Debounce rapid successive writes (e.g. editor save-on-change).
	const debounce = 200 * time.Millisecond
	var timer *time.Timer

	for {
		select {
		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if timer != nil {
					timer.Reset(debounce)
				} else {
					timer = time.AfterFunc(debounce, w.reload)
				}
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("workflow watcher error", "err", err, "path", w.path)
		}
	}
}

func (w *Watcher) reload() {
	def, err := Load(w.path)
	if err != nil {
		// Keep last good config; emit operator-visible error (SPEC.md §6.2).
		slog.Error("workflow reload failed, keeping last good config",
			"err", err, "path", w.path)
		return
	}
	w.mu.Lock()
	w.last = def
	w.mu.Unlock()
	w.onChange(def)
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}

// Last returns the most recent successfully loaded Definition.
func (w *Watcher) Last() *Definition {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.last
}
