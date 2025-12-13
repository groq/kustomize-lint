package kustomization

import (
	"errors"
	"io"
	"os"

	"github.com/charmbracelet/log"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
)

func parseFluxKustomizations(filePath string, source string) ([]string, error) {
	// #nosec G304 - file is controlled by the application's file walking logic
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Debug("Failed to close file", "file", filePath, "err", closeErr)
		}
	}()

	var paths []string
	decoder := yaml.NewDecoder(file)

	for {
		var doc map[string]any

		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Debug("Unable to read file", "file", filePath, "error", err.Error())
			break
		}

		var kustomization kustomizev1.Kustomization
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(doc, &kustomization); err != nil {
			log.Debug("Unable to read file", "file", filePath, "error", err.Error())
			continue
		}

		if kustomization.Spec.SourceRef.Name != source {
			continue
		}

		if kustomization.Spec.Path != "" {
			paths = append(paths, kustomization.Spec.Path)
		}
	}

	if len(paths) > 0 {
		log.Debug("Found paths in Flux Kustomization", "file", filePath, "path", paths)
	}

	return paths, nil
}
