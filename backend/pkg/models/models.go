package models

import "time"

type Repo struct {
	ID            string
	Name          string
	GitHubRepoURL string
	OrgID         string
	CreatedAt     time.Time
}

type Service struct {
	ID        string
	RepoID    string
	Name      string
	Framework string
	HasMetrics bool
	HasOTel   bool
	CreatedAt time.Time
}

type ToggleSpec struct {
	ID          string
	ServiceID   string
	Environment string
	StackMode   string
	Spec        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
