package registry_test

import (
	"bytes"
	"io"
	"io/ioutil"
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

func TestCreateLayer(t *testing.T) {
	type args struct {
		r    io.Reader
		path string
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "tar.gz",
			args: args{
				// gz magic number
				// https://github.com/h2non/filetype/blob/29039c24a9fbddaf40b7ae847d38f7ceafb94dd0/matchers/archive.go#L96-L99
				r: bytes.NewReader([]byte{0x1f, 0x8b, 0x8}),
			},
			want: 3,
		},
		{
			name: "json",
			args: args{
				r: bytes.NewReader([]byte{'{', '}'}),
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatalf("TempDir: %v", err)
			}
			got, err := registry.CreateLayer(tt.args.r, dir)
			if err != nil {
				t.Fatalf("CreateLayer() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("CreateLayer() int64 got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPickupFileinfo(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "valid",
			wantErr: false,
		},
		{
			name:    "invalid",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatalf("TempDir: %v", err)
			}

			var want string
			if !tt.wantErr {
				f, err := ioutil.TempFile(dir, "")
				if err != nil {
					t.Fatalf("TempFile: %v", err)
				}
				f.Close()
				want = filepath.Base(f.Name())
			}

			got, err := registry.PickupFileinfo(dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("PickupFileinfo() error = %v", err)
			}
			if want != got.Name() {
				t.Fatalf("name want: %v, but got %v", want, got.Name())
			}
		})
	}
}
