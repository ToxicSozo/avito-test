package handlers

import (
	"github.com/ToxicSozo/avito-test/api"
	"github.com/ToxicSozo/avito-test/internal/model"
)

type teamEnvelope struct {
	Team api.Team `json:"team"`
}

type userEnvelope struct {
	User api.User `json:"user"`
}

type pullRequestEnvelope struct {
	PullRequest api.PullRequest `json:"pr"`
}

type reassignResponse struct {
	PullRequest api.PullRequest `json:"pr"`
	ReplacedBy  string          `json:"replaced_by"`
}

type reviewerAssignmentsResponse struct {
	UserID       string                 `json:"user_id"`
	PullRequests []api.PullRequestShort `json:"pull_requests"`
}

func teamPayloadToModel(payload api.Team) model.Team {
	members := make([]model.User, 0, len(payload.Members))
	for _, member := range payload.Members {
		members = append(members, model.User{
			ID:       member.UserId,
			Username: member.Username,
			TeamName: payload.TeamName,
			IsActive: member.IsActive,
		})
	}
	return model.Team{
		Name:    payload.TeamName,
		Members: members,
	}
}

func toAPITeam(team *model.Team) api.Team {
	if team == nil {
		return api.Team{}
	}

	members := make([]api.TeamMember, 0, len(team.Members))
	for _, member := range team.Members {
		members = append(members, api.TeamMember{
			UserId:   member.ID,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}

	return api.Team{
		TeamName: team.Name,
		Members:  members,
	}
}

func toAPIUser(user *model.User) api.User {
	if user == nil {
		return api.User{}
	}
	return api.User{
		UserId:   user.ID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}
}

func toAPIPullRequest(pr *model.PullRequest) api.PullRequest {
	if pr == nil {
		return api.PullRequest{}
	}

	reviewers := pr.AssignedReviewers
	if reviewers == nil {
		reviewers = []string{}
	}

	createdAt := pr.CreatedAt
	dto := api.PullRequest{
		PullRequestId:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorId:          pr.AuthorID,
		Status:            api.PullRequestStatus(pr.Status),
		AssignedReviewers: reviewers,
		CreatedAt:         &createdAt,
	}
	if pr.MergedAt != nil {
		dto.MergedAt = pr.MergedAt
	}
	return dto
}

func toAPIPullRequestShorts(prs []model.PullRequestShort) []api.PullRequestShort {
	result := make([]api.PullRequestShort, 0, len(prs))
	for _, pr := range prs {
		result = append(result, api.PullRequestShort{
			PullRequestId:   pr.ID,
			PullRequestName: pr.Name,
			AuthorId:        pr.AuthorID,
			Status:          api.PullRequestShortStatus(pr.Status),
		})
	}
	return result
}

func newReviewerAssignmentsResponse(userID string, prs []model.PullRequestShort) reviewerAssignmentsResponse {
	return reviewerAssignmentsResponse{
		UserID:       userID,
		PullRequests: toAPIPullRequestShorts(prs),
	}
}

func toAPIAssignmentStats(stats *model.AssignmentStats) api.AssignmentStats {
	resp := api.AssignmentStats{
		ByUser:        []api.AssignmentCountByUser{},
		ByPullRequest: []api.AssignmentCountByPullRequest{},
	}
	if stats == nil {
		return resp
	}

	if len(stats.ByUser) > 0 {
		resp.ByUser = make([]api.AssignmentCountByUser, 0, len(stats.ByUser))
		for _, rec := range stats.ByUser {
			resp.ByUser = append(resp.ByUser, api.AssignmentCountByUser{
				UserId:      rec.UserID,
				Assignments: rec.Assignments,
			})
		}
	}

	if len(stats.ByPullRequest) > 0 {
		resp.ByPullRequest = make([]api.AssignmentCountByPullRequest, 0, len(stats.ByPullRequest))
		for _, rec := range stats.ByPullRequest {
			resp.ByPullRequest = append(resp.ByPullRequest, api.AssignmentCountByPullRequest{
				PullRequestId: rec.PullRequestID,
				Reviewers:     rec.Reviewers,
			})
		}
	}

	return resp
}
