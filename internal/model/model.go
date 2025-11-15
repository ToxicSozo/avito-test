package model

import "time"

// Team aggregates all members.
type Team struct {
	Name    string
	Members []User
}

// User describes a single engineer inside a team.
type User struct {
	ID       string
	Username string
	TeamName string
	IsActive bool
}

// PullRequest mirrors storage record with reviewers already loaded.
type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            string
	CreatedAt         time.Time
	MergedAt          *time.Time
	AssignedReviewers []string
}

// PullRequestShort keeps lightweight projection used in listing endpoints.
type PullRequestShort struct {
	ID       string
	Name     string
	AuthorID string
	Status   string
}

// AssignmentStat summarises reviewer assignment counts per user.
type AssignmentStat struct {
	UserID      string
	Username    string
	TeamName    string
	Assignments int64
}
