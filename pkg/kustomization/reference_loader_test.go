package kustomization

import "testing"

func TestReferenceLoader(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		excludes []string
		wantErr  bool
	}{
		{name: "valid", path: "testdata/valid/"},
		{name: "invalid", path: "testdata/invalid/", wantErr: true},
		{name: "invalid references", path: "testdata/invalid_reference/", wantErr: true},
		{name: "unreferenced file", path: "testdata/unreferenced/", wantErr: true},
		{name: "excludes", path: "testdata/unreferenced/", excludes: []string{"file3.yaml", "file4.yaml"}},
		{name: "inline ignore", path: "testdata/inline_ignore/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewReferenceLoader(tt.excludes...).Validate(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
