package kustomization

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/pkg/loader"
	"sigs.k8s.io/kustomize/api/provider"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type ReferenceLoader struct {
	// Files to exclude from the search
	Excludes []string

	// StrictPathCheck enables strict path checking mode
	StrictPathCheck bool

	// FluxSource enables Flux Kustomization parsing
	FluxSource string

	// All files referenced directly from within a `kustomization.yaml`
	referencedFiles map[string]bool

	// All resources found within the same directories as a `kustomization.yaml`
	allResources map[string]bool

	rf *resmap.Factory
}

func NewReferenceLoader(strictPathCheck bool, fluxSource string, excludes ...string) *ReferenceLoader {
	return &ReferenceLoader{
		Excludes:        excludes,
		StrictPathCheck: strictPathCheck,
		FluxSource:      fluxSource,
		referencedFiles: make(map[string]bool),
		allResources:    make(map[string]bool),
		rf:              resmap.NewFactory(provider.NewDepProvider().GetResourceFactory()),
	}
}

// hasInlineIgnore checks if a file contains an inline ignore comment
func hasInlineIgnore(filepath string) bool {
	// #nosec G304 - filepath is controlled by the application's file walking logic
	file, err := os.Open(filepath)
	if err != nil {
		return false
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Debug("Failed to close file", "path", filepath, "err", closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() && lineCount < 10 { // only check first 10 lines
		line := strings.TrimSpace(scanner.Text())
		// check for ignore syntax
		if strings.HasPrefix(line, "#") {
			if strings.Contains(line, "kustomize-lint:ignore") {
				return true
			}
		}
		lineCount++
	}
	return false
}

func (l *ReferenceLoader) Validate(root string) error {
	kustomizations := []string{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Base(path) == ".kustomize-lint-ignore" {
			log.Debug("Skipping directory due to ignore file", "path", path)
			l.Excludes = append(l.Excludes, filepath.Join(filepath.Dir(path), "*"))
			return nil
		}

		if slices.Contains(konfig.RecognizedKustomizationFileNames(), filepath.Base(path)) {
			kustomizations = append(kustomizations, path)
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		if l.FluxSource != "" {
			fluxKustomizations, err := parseFluxKustomizations(path, l.FluxSource)
			if err != nil {
				return err
			}

			for _, kustomization := range fluxKustomizations {
				// Kustomization path is relative to the source root, not the file location
				for _, fileName := range konfig.RecognizedKustomizationFileNames() {
					resolvedPath := filepath.Clean(filepath.Join(root, kustomization, fileName))
					if _, err := os.Stat(resolvedPath); err == nil {
						l.referencedFiles[resolvedPath] = true
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Fatal("Failed to find kustomization.yaml files", "path", root, "err", err)
	}

	for _, kustomization := range kustomizations {
		err := l.walk(root, kustomization)
		if err != nil {
			return err
		}
	}

	var errs []error

	log.Debug("All resources", "resources", l.allResources)
	log.Debug("Referenced files", "files", l.referencedFiles)
	log.Debug("Kustomizations", "kustomizations", kustomizations)

	for r := range l.referencedFiles {
		if !l.allResources[r] {
			errs = append(errs, fmt.Errorf("* referenced file %q not found", r))
		}
	}

	for r := range l.allResources {
		if !l.referencedFiles[r] {
			if !slices.Contains(kustomizations, r) {
				errs = append(errs, fmt.Errorf("* resource %q not referenced", r))
				continue
			}

			containedInKustomization := slices.IndexFunc(kustomizations, func(k string) bool {
				baseDir := filepath.Dir(k)
				return k != r && strings.HasPrefix(r, baseDir+string(filepath.Separator))
			})

			if containedInKustomization != -1 {
				log.Debug(
					"Resource not referenced",
					"resource", r,
					"inKustomization", kustomizations[containedInKustomization],
				)
				errs = append(errs, fmt.Errorf("* resource %q not referenced", r))
			}
		}
	}

	return errors.Join(errs...)
}

func (l *ReferenceLoader) excludedPath(baseDir, path string) bool {
	for _, exclude := range l.Excludes {
		if relativePath, err := filepath.Rel(baseDir, path); err == nil {
			if matched, _ := filepath.Match(exclude, relativePath); matched {
				log.Debug("Skipping path", "path", path, "exclude", exclude)
				return true
			}
		}

		if matched, _ := filepath.Match(exclude, path); matched {
			log.Debug("Skipping path", "path", path, "exclude", exclude)
			return true
		}

		if matched, _ := filepath.Match(exclude, filepath.Base(path)); matched {
			log.Debug("Skipping path", "path", path, "exclude", exclude)
			return true
		}
	}

	if hasInlineIgnore(path) {
		log.Debug("Skipping path due to inline ignore", "path", path)
		return true
	}

	return false
}

func (l *ReferenceLoader) walk(baseDir, path string) error {
	dir := filepath.Dir(path)

	if l.excludedPath(baseDir, path) {
		log.Debug("Skipping path due to exclusion", "path", path)
		return nil
	}

	log.Debug("Walking directory", "path", dir)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}

		if d.Type().IsDir() {
			return nil
		}

		if l.excludedPath(baseDir, path) {
			return nil
		}

		l.allResources[path] = true

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk %q: %v", dir, err)
	}

	log.Debug("Found resources", "resources", l.allResources)

	k := &types.Kustomization{}

	// #nosec G304 - filepath is controlled by the application's file walking logic
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %q: %v", path, err)
	}

	if err := k.Unmarshal(contents); err != nil {
		return fmt.Errorf("failed to unmarshal %q: %v", path, err)
	}

	files := k.Resources[:]
	files = append(files, k.Components...)
	files = append(files, k.Crds...)
	files = append(files, k.Generators...)
	files = append(files, k.Transformers...)
	files = append(files, k.Validators...)
	files = append(files, k.Configurations...)

	for _, s := range k.SecretGenerator {
		for _, fileSource := range s.FileSources {
			_, fileSourcePath, err := parseFileSource(fileSource)
			if err != nil {
				return fmt.Errorf("unable to parse file source %q: %v", fileSource, err)
			}
			files = append(files, fileSourcePath)
		}
		files = append(files, s.EnvSources...)
	}

	for _, c := range k.ConfigMapGenerator {
		for _, fileSource := range c.FileSources {
			_, fileSourcePath, err := parseFileSource(fileSource)
			if err != nil {
				return fmt.Errorf("unable to parse file source %q: %v", fileSource, err)
			}
			files = append(files, fileSourcePath)
		}
		files = append(files, c.EnvSources...)
	}

	for _, p := range k.Patches {
		if p.Path != "" {
			files = append(files, p.Path)
		}
	}

	log.Debug("Found files in path", "path", path, "files", files)

	for _, file := range files {
		resource := (&resource.Origin{Path: dir}).Append(file)

		if resource.Repo != "" {
			ldr := loader.NewFileLoaderAtCwd(filesys.MakeFsOnDisk())
			ll, err := ldr.New(file)
			if err != nil {
				return fmt.Errorf("failed to load %q: %v", file, err)
			}
			if err := ll.Cleanup(); err != nil {
				return fmt.Errorf("failed to cleanup %q: %v", file, err)
			}
		} else if stat, err := os.Stat(resource.Path); err == nil {
			// fast fail on the supplied filepath not matching the cleaned filepath if strict path checking is enabled
			// this can occur as a result of invalid whitespace in the kustomization.yaml
			// https://github.com/kubernetes-sigs/kustomize/issues/5979
			if l.StrictPathCheck && file != filepath.Clean(file) {
				return fmt.Errorf("path %q does not match cleaned path '%s' in %s", file, filepath.Clean(file), path)
			}

			if stat.IsDir() {
				log.Debug("Walking directory", "path", resource.Path)

				p := filepath.Join(resource.Path, "kustomization.yaml")

				if l.referencedFiles[p] {
					continue
				}

				l.referencedFiles[p] = true

				err := l.walk(baseDir, p)
				if err != nil {
					return fmt.Errorf("failed to load kustomization %q: %v", resource.Path, err)
				}
			} else {
				l.referencedFiles[resource.Path] = true
			}
		} else {
			_, err := l.rf.NewResMapFromBytes([]byte(file))
			if err != nil {
				return fmt.Errorf("reference %q cannot be loaded and does not look like YAML: %v", file, err)
			} else {
				log.Debug("Skipping path, looks like YAML", "path", file, "err", err)
			}
		}
	}

	return nil
}
