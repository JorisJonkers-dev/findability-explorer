package main

import (
	"context"
	"errors"

	analysisapp "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	companiesapp "github.com/JorisJonkers-dev/findability-explorer/internal/companies/app"
	prompttestapp "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	prompttestdomain "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// candidateRecorder hands the companies context the prompt test's competitor candidates, to
// record one as added when its company is added from a list.
type candidateRecorder struct {
	prompts *prompttestapp.Service
}

func (c candidateRecorder) RecordAdded(ctx context.Context, user, candidateID, companyID string) error {
	err := c.prompts.RecordAdded(ctx, user, candidateID, companyID)
	if errors.Is(err, prompttestdomain.ErrNotFound) || errors.Is(err, prompttestapp.ErrDecided) {
		return companiesapp.ErrCandidateGone
	}
	return err
}

// scanner hands the companies context the analysis context's scans, to scan companies just added.
type scanner struct {
	analysis *analysisapp.Service
}

func (s scanner) Scan(ctx context.Context, user string, ids []string) (int, error) {
	queued, _, err := s.analysis.Rescan(ctx, user, analysisapp.Pick{IDs: ids}, 0)
	return len(queued), err
}
