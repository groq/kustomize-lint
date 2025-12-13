package kustomization

import "testing"

func TestReferenceLoader(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		excludes        []string
		strictPathCheck bool
		fluxSource      string
		wantErr         bool
	}{
		{name: "valid", path: "testdata/valid/"},
		{name: "invalid", path: "testdata/invalid/", wantErr: true},
		{name: "invalid references", path: "testdata/invalid_reference/", wantErr: true},
		{name: "unreferenced file", path: "testdata/unreferenced/", wantErr: true},
		{name: "unreferenced directory", path: "testdata/unreferenced_dir/", wantErr: true},
		{name: "unreferenced directory with ignore", path: "testdata/unreferenced_dir_ignore/", wantErr: false},
		{name: "excludes", path: "testdata/unreferenced/", excludes: []string{"file3.yaml", "file4.yaml"}},
		{name: "inline ignore", path: "testdata/inline_ignore/"},
		{name: "strict paths enabled", path: "testdata/strict_paths/", strictPathCheck: true, wantErr: true},
		{name: "strict paths disabled", path: "testdata/strict_paths/", strictPathCheck: false, wantErr: false},
		{name: "KV file sources", path: "testdata/kv_file_sources/", wantErr: false},
		{name: "flux without source", path: "testdata/flux/", wantErr: true},
		{name: "flux with source", path: "testdata/flux/", fluxSource: "gitops", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewReferenceLoader(tt.strictPathCheck, tt.fluxSource, tt.excludes...).Validate(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
