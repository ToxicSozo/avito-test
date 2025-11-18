package service

import "errors"

var (
	ErrUserExists        = errors.New("user already exists")
	ErrPullRequestExists = errors.New("pull request already exists")
	ErrPullRequestMerged = errors.New("pull request already merged")
	ErrNotFound          = errors.New("resource not found")
	ErrReviewerMissing   = errors.New("reviewer is not assigned to this pull request")
	ErrNoCandidate       = errors.New("no available replacement candidate")
)
