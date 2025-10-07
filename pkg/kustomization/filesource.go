package kustomization

import (
	"fmt"
	"path"
	"strings"
)

// Copy of kustomize#generators.ParseFileSource
// https://github.com/kubernetes-sigs/kustomize/blob/kustomize/v5.7.1/api/internal/generators/utils.go
func parseFileSource(source string) (keyName, filePath string, err error) {
	numSeparators := strings.Count(source, "=")
	switch {
	case numSeparators == 0:
		return path.Base(source), source, nil
	case numSeparators == 1 && strings.HasPrefix(source, "="):
		return "", "", fmt.Errorf("missing key name for file path %q in source %q", strings.TrimPrefix(source, "="), source)
	case numSeparators == 1 && strings.HasSuffix(source, "="):
		return "", "", fmt.Errorf("missing file path for key name %q in source %q", strings.TrimSuffix(source, "="), source)
	case numSeparators > 1:
		return "", "", fmt.Errorf("source %q key name or file path contains '='", source)
	default:
		components := strings.Split(source, "=")
		return components[0], components[1], nil
	}
}
