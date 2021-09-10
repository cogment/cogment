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
	grpcapi "github.com/cogment/cogment-activity-logger/grpcapi/cogment/api"
)

// TrialStatus represents the stats of a trial
type TrialStatus int

const (
	// TrialNotStarted represents a trial that has yet to start
	TrialNotStarted TrialStatus = iota
	// TrialRunning represents a trial that is currently running
	TrialRunning
	// TrialEnded represents a trial that has ended
	TrialEnded
)

// TrialInfo represents the storage status of the samples in a trial.
type TrialInfo struct {
	TrialID            string
	Status             TrialStatus
	SamplesCount       int
	StoredSamplesCount int
}

// TrialParamsResult represents the result of a query to retrieve trial params
type TrialParamsResult struct {
	Ok  *grpcapi.TrialParams
	Err error
}

// TrialParamsPromise represents the async result of a query to retrieve trial params
type TrialParamsPromise <-chan TrialParamsResult

// DatalogSampleResult represents the result of a query to retrieve a sample
type DatalogSampleResult struct {
	Ok  *grpcapi.DatalogSample
	Err error
}

// DatalogSampleStream represents a stream of sample results
type DatalogSampleStream <-chan DatalogSampleResult

// Backend defines the interface for a datalogger backend
type Backend interface {
	Destroy()

	OnTrialStart(trialID string, trialParams *grpcapi.TrialParams) (TrialInfo, error)
	OnTrialEnd(trialID string) (TrialInfo, error)
	OnTrialSample(trialID string, sample *grpcapi.DatalogSample) (TrialInfo, error)

	GetTrialInfo(trialID string) (TrialInfo, error)
	GetTrialParams(trialID string) (*grpcapi.TrialParams, error)
	GetTrialParamsAsync(trialID string) TrialParamsPromise

	ConsumeTrialSamples(trialID string) DatalogSampleStream
}
