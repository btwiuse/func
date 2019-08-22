package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// A Project is a func project on disk, loaded from .func/project.
type Project struct {
	// RootDir is the absolute path to the root directory of the project.
	RootDir string `json:"-"`

	// Name is the name of the project. If name is modified, Write() should be
	// called to persist the change to disk.
	Name string `json:"name"`
}

// FindProject finds a project on disk. If no project is found, nil is
// returned.
//
// The project's root directory is determined by the file .func/project
// existing. If the given dir does not contain a project, parent directories
// are traversed until a project is found.
func FindProject(dir string) (*Project, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}

	file := filepath.Join(dir, ".func", "project")
	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			parent := filepath.Dir(dir)
			if parent == dir || parent[len(parent)-1] == filepath.Separator {
				// Not found
				return nil, nil
			}
			return FindProject(parent)
		}
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	abs, _ := filepath.Abs(dir)

	d := json.NewDecoder(f)
	d.DisallowUnknownFields()

	p := &Project{RootDir: abs}
	if err := d.Decode(p); err != nil {
		return nil, fmt.Errorf("parse project: %v", err)
	}

	return p, nil
}

// Write persists the project to disk.
func (p *Project) Write() error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(p.RootDir, ".func"), 0744); err != nil {
		return err
	}
	file := filepath.Join(p.RootDir, ".func", "project")
	if err := ioutil.WriteFile(file, b, 0644); err != nil {
		return err
	}
	return nil
}
