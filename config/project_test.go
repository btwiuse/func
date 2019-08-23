package config_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/func/func/config"
	"github.com/google/go-cmp/cmp"
)

func TestFindProject(t *testing.T) {
	abs := func(t *testing.T, dir string) string {
		a, err := filepath.Abs(dir)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}

	tests := []struct {
		name    string
		dir     string
		want    *config.Project
		wantErr bool
	}{
		{"Exact", "testdata/project", &config.Project{
			RootDir: abs(t, "testdata/project"),
			Name:    "testproject",
		}, false},
		{"Subdir", "testdata/project/sub", &config.Project{
			RootDir: abs(t, "testdata/project"),
			Name:    "testproject",
		}, false},
		{"Invalid", "testdata/invalid-project", nil, true},
		{"NoProject", os.TempDir(), nil, false},
		{"NotFound", "nonexisting", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := config.FindProject(tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindRoot() error = %v, wantErr %t", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

func TestProject_create(t *testing.T) {
	dir, done := tempdir(t)
	defer done()

	name := "test-project"
	p := &config.Project{
		Name:    name,
		RootDir: dir,
	}
	err := p.Write()
	if err != nil {
		t.Fatalf("Project.Write() err = %v", err)
	}

	p = readproj(t, dir)
	if p.Name != name {
		t.Errorf("Name does not match; got = %q, want = %q", p.Name, name)
	}

	// Rename
	newname := "foo"
	p.Name = newname
	if err := p.Write(); err != nil {
		t.Fatalf("Write() err = %v", err)
	}

	p = readproj(t, dir)
	if p.Name != newname {
		t.Errorf("Updated name does not match; got = %q, want = %q", p.Name, newname)
	}
}

func tempdir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := ioutil.TempDir("", "functest")
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	}
	return dir, cleanup
}

func readproj(t *testing.T, dir string) *config.Project {
	t.Helper()
	p, err := config.FindProject(dir)
	if err != nil {
		t.Fatalf("FindProject() err = %v", err)
	}
	if p == nil {
		t.Fatal("Project not found")
	}
	return p
}
