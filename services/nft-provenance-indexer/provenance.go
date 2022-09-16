package main

import (
	"time"

	"go.uber.org/cadence/workflow"
)

type Provenance struct {
	indexID    string
	provenance []string
	owners     string
	fungible   string
	assetID    string
}

func (p *Provenance) GenerateProvenance(ctx workflow.Context, tokenOwner string) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	log := workflow.GetLogger(ctx)

	return error
}
