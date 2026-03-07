package workspace

import (
	"os"
	"path/filepath"
)

type Workspace struct {
	Path         string
	WorkspaceKey string
	CreatedNow   bool
}

func EnsureWorkspace(root, identifier string) (Workspace, error) {
	key := SanitizeKey(identifier)
	path := filepath.Join(root, key)
	if err := ValidatePath(root, path); err != nil {
		return Workspace{}, err
	}

	_, err := os.Stat(path)
	switch {
	case err == nil:
		return Workspace{Path: path, WorkspaceKey: key, CreatedNow: false}, nil
	case !os.IsNotExist(err):
		return Workspace{}, err
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return Workspace{}, err
	}
	return Workspace{Path: path, WorkspaceKey: key, CreatedNow: true}, nil
}
