// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend

import (
	"context"
	"fmt"

	grpcapi "github.com/cogment/cogment-trial-datastore/grpcapi/cogment/api"
)

// TrialInfo represents the storage status of the samples in a trial.
type TrialInfo struct {
	TrialID            string
	UserID             string
	State              grpcapi.TrialState
	SamplesCount       int
	StoredSamplesCount int
}

type TrialsInfoResult struct {
	TrialInfos   []*TrialInfo
	NextTrialIdx int
}

type TrialsInfoObserver chan TrialsInfoResult

// TrialParams represents the params of a trials
type TrialParams struct {
	TrialID string
	UserID  string
	Params  *grpcapi.TrialParams
}

// TrialSampleFilter represents the arguments that can be passed to create or update a trial
type TrialSampleFilter struct {
	TrialIDs             []string
	ActorNames           []string
	ActorClasses         []string
	ActorImplementations []string
	Fields               []grpcapi.StoredTrialSampleField
}

type TrialSampleObserver chan *grpcapi.StoredTrialSample

// Backend defines the interface for a datalogger backend
type Backend interface {
	Destroy()

	CreateOrUpdateTrials(ctx context.Context, trialsParams []*TrialParams) error
	RetrieveTrials(ctx context.Context, filter []string, fromTrialIdx int, count int) (TrialsInfoResult, error)
	ObserveTrials(ctx context.Context, filter []string, fromTrialIdx int, count int, out chan<- TrialsInfoResult) error
	DeleteTrials(ctx context.Context, trialIDs []string) error

	GetTrialParams(ctx context.Context, trialIDs []string) ([]*TrialParams, error)

	AddSamples(ctx context.Context, samples []*grpcapi.StoredTrialSample) error
	ObserveSamples(ctx context.Context, filter TrialSampleFilter, out chan<- *grpcapi.StoredTrialSample) error
}

// UnknownTrialError is raised when trying to operate on an unknown trial
type UnknownTrialError struct {
	TrialID string
}

func (e *UnknownTrialError) Error() string {
	return fmt.Sprintf("no trial %q found", e.TrialID)
}
