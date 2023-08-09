// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package actor

import (
	"errors"
	"fmt"
)

// TrialLeftError is raised when the actor left the trial
type TrialLeftError struct {
	TrialID   string
	ActorName string
}

func NewTrialLeftError(trialID string, actorName string) *TrialLeftError {
	return &TrialLeftError{TrialID: trialID, ActorName: actorName}
}

func (err *TrialLeftError) Error() string {
	return fmt.Sprintf("Trial [%s] was left by actor [%s]", err.TrialID, err.ActorName)
}

// TrialEndedError is raised when the trial was ended
type TrialEndedError struct {
	TrialID string
}

func NewTrialEndedError(trialID string) *TrialEndedError {
	return &TrialEndedError{TrialID: trialID}
}

func (err *TrialEndedError) Error() string {
	return fmt.Sprintf("Trial [%s] ended", err.TrialID)
}

// TrialNotFoundError is raised when the trial is not found for a given actor
type TrialNotFoundError struct {
	TrialID   string
	ActorName string
}

func NewTrialNotFoundError(trialID string, actorName string) *TrialNotFoundError {
	return &TrialNotFoundError{TrialID: trialID, ActorName: actorName}
}

func (err *TrialNotFoundError) Error() string {
	return fmt.Sprintf("Trial [%s] for actor [%s] not found", err.TrialID, err.ActorName)
}

// TrialTransitionError is raised when an invalid trial status transition is requested
type TrialTransitionError struct {
	TrialID         string
	ActorName       string
	Status          actorTrialStatus
	TickID          uint64
	RequestedStatus actorTrialStatus
	RequestedTickID uint64
}

func NewTrialTransitionError(
	trialID string,
	actorName string,
	status actorTrialStatus,
	requestedStatus actorTrialStatus,

) *TrialTransitionError {
	return &TrialTransitionError{
		TrialID:         trialID,
		ActorName:       actorName,
		Status:          status,
		TickID:          0,
		RequestedStatus: requestedStatus,
		RequestedTickID: 0,
	}
}

func NewRunningTrialTransitionError(
	trialID string,
	actorName string,
	status actorTrialStatus,
	tickID uint64,
	requestedStatus actorTrialStatus,
	requestedTickID uint64,
) *TrialTransitionError {
	return &TrialTransitionError{
		TrialID:         trialID,
		ActorName:       actorName,
		Status:          status,
		TickID:          tickID,
		RequestedStatus: requestedStatus,
		RequestedTickID: requestedTickID,
	}
}

func (err *TrialTransitionError) Error() string {
	if err.Status == waitForRecvEvent && err.RequestedStatus == waitForSentEvent {
		return fmt.Sprintf(
			"Trial [%s] for actor [%s] is at tick [%d], it can't receive event for tick [%d]",
			err.TrialID,
			err.ActorName,
			err.TickID,
			err.RequestedTickID,
		)
	}
	if err.Status == waitForSentEvent && err.RequestedStatus == waitForRecvEvent {
		return fmt.Sprintf(
			"Trial [%s] for actor [%s] is at tick [%d], it can't send event for tick [%d]",
			err.TrialID,
			err.ActorName,
			err.TickID,
			err.RequestedTickID,
		)
	}
	return fmt.Sprintf(
		"Trial [%s] for actor [%s] can't transition from status [%s] to [%s]",
		err.TrialID,
		err.ActorName,
		err.Status,
		err.RequestedStatus,
	)
}

// ErrUnexpected is raised when something unexpected occur, it shouldn't be raised by users input
var ErrUnexpected = errors.New("Unexpected Error")
