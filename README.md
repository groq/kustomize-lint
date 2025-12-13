## kustomize file and directory linter

[kustomize](https://github.com/kubernetes-sigs/kustomize) does not allow wildcard inclusion of files within a directory nor does it provide a "strict" option to disallow and detect unreferenced or extraneous files.

This repository is generically named, but currently provides a single linting rule: detect files that not referenced in any `kustomization.yaml` configurable and to make sure all referenced files exist.

### Usage

For basic usage, provide the root path containing any number of kustomizations.

```sh
$ kustomize-lint lint path/to/root
```

For example:
```sh
$ tree path/to/root
├── base
│   ├── file.yaml
│   ├── file2.yaml
│   └── kustomization.yaml
└── overlay
    ├── file3.yaml
    ├── kustomization.yaml

$ cat base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - file.yaml

$ cat overlay/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../base
```

Running the linter will provide errors for `file2.yaml` and `file3.yaml`:
```sh
$ kustomize-lint lint path/to/root 
FATA Validation errors
  err=
  │ * resource "path/to/root/base/file2.yaml" not referenced
  │ * resource "path/to/root/overlay/file3.yaml" not referenced
exit status 1
```

The linter will also error for referenced files that do not exist:
```sh
$ rm base/file.yaml

$ kustomize-lint lint path/to/root
FATA Validation errors err="reference \"file.yaml\" cannot be loaded and does not look like YAML: missing Resource metadata"
exit status 1
```

#### Ignoring Files

To explicitly ignore files that are not referenced, the `--exclude` (`-x`) flag can be provided or an inline `# kustomize-lint:ignore` comment can be added to the file.

For example, with this directory structure:
```sh
$ cat kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - file.yaml
  # - ignored_file.yaml
$ ls
file.yaml  ignored_file.yaml  kustomization.yaml
```

The lint will fail:
```sh
$ kustomize-lint .
FATA Validation errors err="* resource \"ignored_file.yaml\" not referenced"
```

Exclude it with a command-line flag:
```sh
$ kustomize-lint -x ignored_file.yaml
```

Or, by adding the `# kustomize-lint:ignore` comment within the first 10 lines of the file:
```sh
$ head ignored_file.yaml
# kustomize-lint:ignore
# This file is temporarily disabled but we want to keep it in the repo
---
apiVersion: v1
kind: ConfigMap
```

#### Ignoring Directories

To explicitly ignore directories that are not referenced, the `--exclude` (`-x`) flag can be provided or create an empty `.kustomize-lint-ignore` file within the directory.

For example, with this directory structure:
```sh
$ cat kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  # - ignored_dir
$ ls
ignored_dir  kustomization.yaml

$ cat ignored_dir/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - file.yaml
$ ls ignored_dir
file.yaml  kustomization.yaml
```

The lint will fail:
```sh
$ kustomize-lint .
FATA Validation errors err="* resource \"ignored_dir/kustomization.yaml\" not referenced"
```

Exclude it with a command-line flag:
```sh
$ kustomize-lint -x 'ignored_dir/*'
```

Or, by adding the `.kustomize-lint-ignore` file:
```sh
$ touch ignored_dir/.kustomize-lint-ignore
```

#### Strict File Checking

To workaround [kubernetes/kustomize#5979](https://github.com/kubernetes-sigs/kustomize/issues/5979), the `--strict-path-check` (`-s`) flag will fail if a file reference does not match the output of [`filepath.Clean`](https://pkg.go.dev/path/filepath#Clean).

#### Debugging

To output more information, provide the `--debug` flag:
```sh
$ kustomize-lint --debug lint path/to/root
```

### Contributing

Any contributions you make are greatly appreciated. If you have a suggestion that would make this better, please fork the repo and create a pull request.

See the [contributing documentation](./CONTRIBUTING.md) for more information.
