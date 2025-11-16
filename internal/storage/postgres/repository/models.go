package repository

import (
	"time"

	"github.com/ToxicSozo/avito-test/internal/model"
)

type Team struct {
	Name    string `gorm:"column:team_name;primaryKey"`
	Members []User `gorm:"foreignKey:TeamName;references:Name"`
}

func (Team) TableName() string {
	return "teams"
}

type User struct {
	ID       string `gorm:"column:user_id;primaryKey"`
	Username string `gorm:"column:username"`
	TeamName string `gorm:"column:team_name"`
	IsActive bool   `gorm:"column:is_active"`
}

func (User) TableName() string {
	return "users"
}

type PullRequest struct {
	ID        string                `gorm:"column:pull_request_id;primaryKey"`
	Name      string                `gorm:"column:pull_request_name"`
	AuthorID  string                `gorm:"column:author_id"`
	Status    string                `gorm:"column:status"`
	CreatedAt time.Time             `gorm:"column:created_at"`
	MergedAt  *time.Time            `gorm:"column:merged_at"`
	Reviewers []PullRequestReviewer `gorm:"foreignKey:PullRequestID;references:ID"`
}

func (PullRequest) TableName() string {
	return "pull_requests"
}

type PullRequestReviewer struct {
	PullRequestID string    `gorm:"column:pull_request_id;primaryKey"`
	ReviewerID    string    `gorm:"column:reviewer_id"`
	Position      int       `gorm:"column:position;primaryKey"`
	AssignedAt    time.Time `gorm:"column:assigned_at"`
}

func (PullRequestReviewer) TableName() string {
	return "pull_request_reviewers"
}

func (t Team) toModel() *model.Team {
	members := make([]model.User, 0, len(t.Members))
	for _, m := range t.Members {
		members = append(members, m.toModel())
	}
	return &model.Team{
		Name:    t.Name,
		Members: members,
	}
}

func (u User) toModel() model.User {
	return model.User{
		ID:       u.ID,
		Username: u.Username,
		TeamName: u.TeamName,
		IsActive: u.IsActive,
	}
}

func userFromModel(mu model.User) User {
	return User{
		ID:       mu.ID,
		Username: mu.Username,
		TeamName: mu.TeamName,
		IsActive: mu.IsActive,
	}
}

func (pr PullRequest) toModel() *model.PullRequest {
	reviewers := make([]string, 0, len(pr.Reviewers))
	for _, r := range pr.Reviewers {
		reviewers = append(reviewers, r.ReviewerID)
	}
	return &model.PullRequest{
		ID:                pr.ID,
		Name:              pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            pr.Status,
		CreatedAt:         pr.CreatedAt,
		MergedAt:          pr.MergedAt,
		AssignedReviewers: reviewers,
	}
}
