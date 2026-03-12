package project

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ahtwr/cw/internal/config"
	"github.com/ahtwr/cw/internal/git"
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
	dir := config.ProjectsDir()
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
	dir := filepath.Join(config.ProjectsDir(), name)
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
	dir := filepath.Join(config.ProjectsDir(), name)
	err := os.MkdirAll(dir, 0755)
	return dir, err
}

func Rename(oldName, newName string) error {
	oldPath := filepath.Join(config.ProjectsDir(), oldName)
	newPath := filepath.Join(config.ProjectsDir(), newName)
	return os.Rename(oldPath, newPath)
}

func Delete(name string) error {
	dir := filepath.Join(config.ProjectsDir(), name)
	return os.RemoveAll(dir)
}

func HasClaudeMD(name string) bool {
	dir := filepath.Join(config.ProjectsDir(), name)
	_, err := os.Stat(filepath.Join(dir, "CLAUDE.md"))
	return err == nil
}

func RemoveRepo(projectName, repoName string) error {
	repoPath := filepath.Join(config.ProjectsDir(), projectName, repoName)
	return os.RemoveAll(repoPath)
}

func CopyDir(src, dest string) error {
	cmd := exec.Command("cp", "-r", src, dest)
	return cmd.Run()
}

func MoveDir(src, dest string) error {
	return os.Rename(src, dest)
}
