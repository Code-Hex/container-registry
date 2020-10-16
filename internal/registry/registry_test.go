package registry_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Code-Hex/container-registry/internal/registry"
)

func TestMain(m *testing.M) {
	registry.BasePath = "testdata"
	os.Exit(m.Run())
}

func TestPathJoinWithBase(t *testing.T) {
	type args struct {
		name string
		p    []string
	}
	tests := []struct {
		name     string
		args     args
		basePath string
		want     string
	}{
		{
			name: "simple",
			args: args{
				name: "library/hello-world",
				p: []string{
					"digest",
					"layer.tar.gz",
				},
			},
			basePath: "base",
			want:     filepath.Join("base", "library/hello-world", "digest", "layer.tar.gz"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry.BasePath = tt.basePath
			if got := registry.PathJoinWithBase(tt.args.name, tt.args.p...); got != tt.want {
				t.Errorf("PathJoinWithBase() = %v, want %v", got, tt.want)
			}
		})
	}
}
