package kustomization

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
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

	// All files referenced directly from within a `kustomization.yaml`
	referencedFiles map[string]bool

	// All resources found within the same directories as a `kustomization.yaml`
	allResources map[string]bool

	rf *resmap.Factory
}

func NewReferenceLoader(excludes ...string) *ReferenceLoader {
	return &ReferenceLoader{
		Excludes:        excludes,
		referencedFiles: make(map[string]bool),
		allResources:    make(map[string]bool),
		rf:              resmap.NewFactory(provider.NewDepProvider().GetResourceFactory()),
	}
}

func (l *ReferenceLoader) Validate(path string) error {
	kustomizations := []string{}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Base(path) == "kustomization.yaml" {
			kustomizations = append(kustomizations, path)
		}

		return nil
	})
	if err != nil {
		log.Fatal("Failed to find kustomization.yaml files", "path", path, "err", err)
	}

	for _, kustomization := range kustomizations {
		err := l.walk(path, kustomization)
		if err != nil {
			return err
		}
	}

	var errs []error

	log.Debug("All resources", "resources", l.allResources)
	log.Debug("Referenced files", "files", l.referencedFiles)

	for r := range l.referencedFiles {
		if !l.allResources[r] {
			errs = append(errs, fmt.Errorf("* referenced file %q not found", r))
		}
	}

	for r := range l.allResources {
		if !l.referencedFiles[r] {
			errs = append(errs, fmt.Errorf("* resource %q not referenced", r))
		}
	}

	return errors.Join(errs...)
}

func (l *ReferenceLoader) walk(baseDir, path string) error {
	dir := filepath.Dir(path)

	l.referencedFiles[path] = true

	log.Debug("Walking directory", "path", dir)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.Type().IsDir() {
			return nil
		}

		for _, exclude := range l.Excludes {
			if relativePath, err := filepath.Rel(baseDir, path); err == nil {
				if matched, _ := filepath.Match(exclude, relativePath); matched {
					log.Debug("Skipping path", "path", path, "exclude", exclude)
					return nil
				}
			}

			if matched, _ := filepath.Match(exclude, path); matched {
				log.Debug("Skipping path", "path", path, "exclude", exclude)
				return nil
			}

			if matched, _ := filepath.Match(exclude, filepath.Base(path)); matched {
				log.Debug("Skipping path", "path", path, "exclude", exclude)
				return nil
			}
		}

		l.allResources[path] = true

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk %q: %v", dir, err)
	}

	log.Debug("Found resources", "resources", l.allResources)

	k := &types.Kustomization{}

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
		files = append(files, s.GeneratorArgs.FileSources...)
		files = append(files, s.GeneratorArgs.EnvSources...)
	}

	for _, c := range k.ConfigMapGenerator {
		files = append(files, c.GeneratorArgs.FileSources...)
		files = append(files, c.GeneratorArgs.EnvSources...)
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
			if stat.IsDir() {
				log.Debug("Walking directory", "path", resource.Path)

				p := filepath.Join(resource.Path, "kustomization.yaml")

				if l.referencedFiles[p] {
					continue
				}

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
