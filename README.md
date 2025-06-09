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
```
$ rm base/file.yaml

$ kustomize-lint lint path/to/root
FATA Validation errors err="reference \"file.yaml\" cannot be loaded and does not look like YAML: missing Resource metadata"
exit status 1
```

#### Debugging

To output more information, provide the `--debug` flag:
```sh
$ kustomize-lint --debug lint path/to/root
```

### Contributing

Any contributions you make are greatly appreciated. If you have a suggestion that would make this better, please fork the repo and create a pull request.

See the [contributing documentation](./CONTRIBUTING.md) for more information.
