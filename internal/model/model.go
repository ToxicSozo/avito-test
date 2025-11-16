package model

import "time"

type Team struct {
	Name    string
	Members []User
}

type User struct {
	ID       string
	Username string
	TeamName string
	IsActive bool
}

type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            string
	CreatedAt         time.Time
	MergedAt          *time.Time
	AssignedReviewers []string
}

type PullRequestShort struct {
	ID       string
	Name     string
	AuthorID string
	Status   string
}

type AssignmentStat struct {
	UserID      string
	Username    string
	TeamName    string
	Assignments int64
}
