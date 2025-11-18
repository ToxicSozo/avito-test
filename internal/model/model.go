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

type AssignmentCountByUser struct {
	UserID      string
	Assignments int64
}

type AssignmentCountByPullRequest struct {
	PullRequestID string
	Reviewers     int64
}

type AssignmentStats struct {
	ByUser        []AssignmentCountByUser
	ByPullRequest []AssignmentCountByPullRequest
}
