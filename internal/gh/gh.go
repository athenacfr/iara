package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type Org struct {
	Login string `json:"login"`
}

type Repo struct {
	Name            string   `json:"name"`
	NameWithOwner   string   `json:"nameWithOwner"`
	Owner           string   `json:"-"`
	Description     string   `json:"description"`
	PrimaryLanguage Language `json:"primaryLanguage"`
	StargazerCount  int      `json:"stargazerCount"`
	UpdatedAt       string   `json:"updatedAt"`
}

type Language struct {
	Name string `json:"name"`
}

type RepoGroup struct {
	Owner string
	Repos []Repo
}

func IsAvailable() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

func IsAuthenticated() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}

func Username() string {
	out, err := run("api", "user", "--jq", ".login")
	if err != nil {
		return "personal"
	}
	name := strings.TrimSpace(out)
	if name == "" {
		return "personal"
	}
	return name
}

func ListOrgs() ([]Org, error) {
	out, err := run("api", "user/orgs", "--jq", ".[].login")
	if err != nil {
		return nil, err
	}

	var orgs []Org
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			orgs = append(orgs, Org{Login: line})
		}
	}
	return orgs, nil
}

func ListRepos(org string) ([]Repo, error) {
	args := []string{"repo", "list"}
	if org != "" {
		args = append(args, org)
	}
	args = append(args,
		"--json", "name,nameWithOwner,description,primaryLanguage,stargazerCount,updatedAt",
		"--limit", "1000",
	)

	out, err := run(args...)
	if err != nil {
		return nil, err
	}

	var repos []Repo
	if err := json.Unmarshal([]byte(out), &repos); err != nil {
		return nil, fmt.Errorf("parsing repo list: %w", err)
	}
	return repos, nil
}

func FetchRepos() ([]RepoGroup, error) {
	username := Username()

	owners := []string{username}

	orgs, err := ListOrgs()
	if err != nil {
		return nil, fmt.Errorf("listing orgs: %w", err)
	}
	for _, org := range orgs {
		owners = append(owners, org.Login)
	}

	var groups []RepoGroup
	for _, owner := range owners {
		repos, err := ListRepos(owner)
		if err != nil {
			continue
		}
		for i := range repos {
			repos[i].Owner = owner
		}
		sort.Slice(repos, func(i, j int) bool {
			return strings.ToLower(repos[i].Name) < strings.ToLower(repos[j].Name)
		})
		groups = append(groups, RepoGroup{
			Owner: owner,
			Repos: repos,
		})
	}

	return groups, nil
}

func CloneRepo(fullName, destPath string) error {
	cmd := exec.Command("gh", "repo", "clone", fullName, destPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func run(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gh %s: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return "", err
	}
	return string(out), nil
}
