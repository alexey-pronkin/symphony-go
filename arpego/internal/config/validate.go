package config

import "fmt"

// ValidationError categories per SPEC.md §11.4 and §6.3.
const (
	ErrUnsupportedTrackerKind         = "unsupported_tracker_kind"
	ErrUnsupportedTrackerStorage      = "unsupported_tracker_storage"
	ErrUnsupportedRuntimeStateStorage = "unsupported_runtime_state_storage"
	ErrMissingTrackerAPIKey           = "missing_tracker_api_key"
	ErrMissingTrackerProjectSlug      = "missing_tracker_project_slug"
	ErrMissingPostgresDSN             = "missing_postgres_dsn"
	ErrMissingCodexCommand            = "missing_codex_command"
)

// ValidationError is a typed config validation failure.
type ValidationError struct {
	Kind    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

// ValidateDispatch performs dispatch preflight validation (SPEC.md §6.3).
// It validates the fields required to poll and launch workers.
func ValidateDispatch(c Config) error {
	kind := c.TrackerKind()
	if kind == "" {
		return &ValidationError{Kind: ErrUnsupportedTrackerKind, Message: "tracker.kind is missing"}
	}
	if kind != "linear" && kind != "local" {
		return &ValidationError{
			Kind:    ErrUnsupportedTrackerKind,
			Message: fmt.Sprintf("unsupported tracker kind: %q (supported: linear, local)", kind),
		}
	}
	if kind == "linear" && c.TrackerAPIKey() == "" {
		return &ValidationError{
			Kind:    ErrMissingTrackerAPIKey,
			Message: "tracker.api_key is missing or resolved to empty string",
		}
	}
	if kind == "linear" && c.TrackerProjectSlug() == "" {
		return &ValidationError{
			Kind:    ErrMissingTrackerProjectSlug,
			Message: "tracker.project_slug is required for tracker.kind=linear",
		}
	}
	if kind == "local" {
		storage := c.TrackerStorage()
		if storage != "file" && storage != "postgres" {
			return &ValidationError{
				Kind:    ErrUnsupportedTrackerStorage,
				Message: fmt.Sprintf("unsupported tracker.storage: %q (supported: file, postgres)", storage),
			}
		}
		if storage == "postgres" && c.StoragePostgresDSN() == "" {
			return &ValidationError{
				Kind:    ErrMissingPostgresDSN,
				Message: "storage.postgres_dsn or SYMPHONY_POSTGRES_DSN is required for tracker.storage=postgres",
			}
		}
	}
	if storage := c.StorageRuntimeState(); storage != "" {
		if storage != "postgres" {
			return &ValidationError{
				Kind:    ErrUnsupportedRuntimeStateStorage,
				Message: fmt.Sprintf("unsupported storage.runtime_state: %q (supported: postgres)", storage),
			}
		}
		if c.StoragePostgresDSN() == "" {
			return &ValidationError{
				Kind:    ErrMissingPostgresDSN,
				Message: "storage.postgres_dsn or SYMPHONY_POSTGRES_DSN is required for storage.runtime_state=postgres",
			}
		}
	}
	if c.CodexCommand() == "" {
		return &ValidationError{
			Kind:    ErrMissingCodexCommand,
			Message: "codex.command must not be empty",
		}
	}
	return nil
}
