package workspace

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var invalidWorkspaceChars = regexp.MustCompile(`[^A-Za-z0-9._-]`)

func SanitizeKey(id string) string {
	return invalidWorkspaceChars.ReplaceAllString(id, "_")
}

func ValidatePath(root, path string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return fmt.Errorf("compare path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("workspace path %q escapes root %q", absPath, absRoot)
	}
	return nil
}
