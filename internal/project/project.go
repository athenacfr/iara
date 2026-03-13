package project

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ahtwr/cw/internal/git"
	"github.com/ahtwr/cw/internal/paths"
)

type Repo struct {
	Name       string
	Path       string
	Branch     string
	DirtyFiles []string
}

type Project struct {
	Name  string
	Path  string
	Repos []Repo
}

func List() ([]Project, error) {
	dir := paths.ProjectsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var projects []Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := Get(e.Name())
		if err != nil {
			continue
		}
		projects = append(projects, *p)
	}
	return projects, nil
}

func Get(name string) (*Project, error) {
	dir := filepath.Join(paths.ProjectsDir(), name)
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrNotExist
	}

	p := &Project{
		Name: name,
		Path: dir,
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return p, nil
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		repoPath := filepath.Join(dir, e.Name())
		if !git.IsRepo(repoPath) {
			continue
		}

		r := Repo{
			Name:       e.Name(),
			Path:       repoPath,
			Branch:     git.Branch(repoPath),
			DirtyFiles: git.DirtyFiles(repoPath),
		}

		p.Repos = append(p.Repos, r)
	}

	return p, nil
}

func Create(name string) (string, error) {
	dir := filepath.Join(paths.ProjectsDir(), name)
	err := os.MkdirAll(dir, 0755)
	return dir, err
}

func Rename(oldName, newName string) error {
	oldPath := filepath.Join(paths.ProjectsDir(), oldName)
	newPath := filepath.Join(paths.ProjectsDir(), newName)
	return os.Rename(oldPath, newPath)
}

func Delete(name string) error {
	dir := filepath.Join(paths.ProjectsDir(), name)
	return os.RemoveAll(dir)
}

func RemoveRepo(projectName, repoName string) error {
	repoPath := filepath.Join(paths.ProjectsDir(), projectName, repoName)
	return os.RemoveAll(repoPath)
}

func CopyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func MoveDir(src, dest string) error {
	return os.Rename(src, dest)
}
