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
	"context"
	"errors"
	"sync"

	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/cogment/cogment/utils"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	trialSpecManager *trialspec.Manager

	trialsLock       sync.RWMutex
	trials           []*actorTrial
	trialsObservable *utils.Observable
}

func NewManager(trialSpecManager *trialspec.Manager) (*Manager, error) {
	manager := Manager{
		trialSpecManager: trialSpecManager,
		trialsLock:       sync.RWMutex{},
		trials:           []*actorTrial{},
		trialsObservable: utils.NewObservable(),
	}

	return &manager, nil
}

func (manager *Manager) getTrial(trialID string, actorName string) *actorTrial {
	manager.trialsLock.RLock()
	defer manager.trialsLock.RUnlock()
	for _, trial := range manager.trials {
		if trial.Matches(trialID, actorName) {
			return trial
		}
	}
	return nil
}

func (manager *Manager) addTrialIfNeeded(
	trialID string,
	actorName string,
	trialFactory func() *actorTrial,
) (*actorTrial, bool) {
	manager.trialsLock.Lock()
	defer manager.trialsLock.Unlock()
	for _, existingTrial := range manager.trials {
		if existingTrial.Matches(trialID, actorName) {
			return existingTrial, false
		}
	}
	newTrial := trialFactory()
	manager.trials = append(manager.trials, newTrial)
	manager.trialsObservable.Emit()
	return newTrial, true
}

func (manager *Manager) removeTrial(trialID string, actorName string) {
	manager.trialsLock.Lock()
	defer manager.trialsLock.Unlock()
	nbTrials := len(manager.trials)
	for trialIndex, trial := range manager.trials {
		if trial.Matches(trialID, actorName) {
			// Put the last trial instead of this one
			manager.trials[trialIndex] = manager.trials[nbTrials-1]
			// Pop the last trial
			manager.trials = manager.trials[:nbTrials-1]
			manager.trialsObservable.Emit()
			return
		}
	}
}

func (manager *Manager) leaveAllTrials() {
	manager.trialsLock.Lock()
	defer manager.trialsLock.Unlock()
	for _, trial := range manager.trials {
		_ = trial.Leave()
	}
	manager.trialsObservable.Emit()
}

// waitForTrial waits for a trial to satisfy a test function and returns it.
//
// During the waiting period, the test function is called with the trial as argument,
// it can be `nil` if no matching trial exist.
//
// If the test returns true, `waitForTrial` returns the trial.
// If the test returns false, the function waits for a new trial to satisfy it.
// If the test returns an error, `waitForTrial` returns the error.
//
// The given context can be used to cancel the waiting.
func (manager *Manager) waitForTrial(
	ctx context.Context,
	trialID string,
	actorName string,
	trialTest func(trial *actorTrial) (bool, error),
) (*actorTrial, error) {
	observer := manager.trialsObservable.Subscribe()
	defer manager.trialsObservable.Unsubscribe(observer)
	for {
		trial := manager.getTrial(trialID, actorName)
		ok, err := trialTest(trial)
		if err != nil {
			return nil, err
		}
		if ok {
			return trial, nil
		}
		select {
		case <-observer.Receive():
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (manager *Manager) Destroy() {
	log.Debug("Leave all trials")
	manager.leaveAllTrials()
}

func (manager *Manager) Spec() *trialspec.Manager {
	return manager.trialSpecManager
}

// GetTrial retrieves the information about a trial joined (or waiting to be joined) by an actor
func (manager *Manager) GetTrial(trialID string, actorName string) (TrialInfo, error) {
	trial := manager.getTrial(trialID, actorName)
	if trial == nil {
		return TrialInfo{}, NewTrialNotFoundError(trialID, actorName)
	}
	return trial.Info(), nil
}

type JoinTrialResult struct {
	TrialInfo
	RecvEvent
	ActorConfig *trialspec.DynamicPbMessage `json:"actor_config,omitempty"`
}

// JoinTrial awaits for a trial to be started, make an actor join it and wait for the initial received event.
//
// JoinTrial and StartTrial+RequestAct are used in conjunction: JoinTrial is used by the endpoint
// accessed by the actor, StartTrial+RequestAct are used by the orchestrator's.
func (manager *Manager) JoinTrial(ctx context.Context, trialID string, actorName string) (JoinTrialResult, error) {
	log := log.WithFields(logrus.Fields{"trial_id": trialID, "actor_name": actorName})
	log.Debug("Joining trial")

	// 1 - add the actor trial if needed, and mark it as `joined`
	trial, added := manager.addTrialIfNeeded(trialID, actorName, func() *actorTrial {
		return newJoinedActorTrial(trialID, actorName)
	})
	if !added {
		// Actor trial already existed, marking it as `joined`
		err := trial.Join()
		if err != nil {
			var trialTransitionError *TrialTransitionError
			// Support joining twice to account for http / web browser shenanigans
			if !errors.As(err, &trialTransitionError) ||
				(trialTransitionError.Status != joined &&
					trialTransitionError.Status != waitForRecvEvent &&
					trialTransitionError.Status != waitForSentEvent) {
				return JoinTrialResult{}, err
			}
		}
		manager.trialsObservable.Emit()
	}

	// 2 - wait for it to start and have its first observation
	log.Debug("Waiting for trial start and initial observation")
	hasInitialObservation := func(trial *actorTrial) (bool, error) {
		if trial == nil {
			return false, nil
		}
		switch trial.Status() {
		case ended:
			return false, NewTrialEndedError(trialID)
		case left:
			return false, NewTrialLeftError(trialID, actorName)
		case waitForSentEvent:
			return true, nil
		case waitForRecvEvent:
			return false, nil
		case joined:
			return false, nil
		default:
			return false, ErrUnexpected
		}
	}
	trial, err := manager.waitForTrial(ctx, trialID, actorName, hasInitialObservation)
	if err != nil {
		return JoinTrialResult{}, err
	}

	return JoinTrialResult{
		TrialInfo:   trial.Info(),
		ActorConfig: trial.ActorConfig(),
		RecvEvent:   trial.LastRecvEvent(),
	}, nil
}

// StartTrial marks a trial as started and wait for the actor to join it
//
// cf. JoinTrial
func (manager *Manager) StartTrial(
	ctx context.Context,
	trialID string,
	actorName string,
	actorClass string,
	actorImplementation string,
	actorConfig *trialspec.DynamicPbMessage,
) error {
	log := log.WithFields(logrus.Fields{"trial_id": trialID, "actor_name": actorName})
	log.Debug("Starting trial")

	// 1 - add the actor trial if needed, and mark it as `started`
	trial, added := manager.addTrialIfNeeded(trialID, actorName, func() *actorTrial {
		return newStartedActorTrial(
			trialID,
			actorName,
			actorClass,
			actorImplementation,
			actorConfig,
		)
	})
	if !added {
		// Actor trial already existed, marking it as `started`
		err := trial.Start(
			actorClass,
			actorImplementation,
			actorConfig,
		)
		if err != nil {
			return err
		}
		manager.trialsObservable.Emit()
		// If `start` succeeded the actor trial is already joined!
		return nil
	}

	// 2 - wait for it to be joined
	log.Debug("Waiting for trial to be joined")
	isJoined := func(trial *actorTrial) (bool, error) {
		if trial == nil {
			return false, nil
		}
		switch trial.Status() {
		case ended:
			return false, NewTrialEndedError(trialID)
		case left:
			return false, NewTrialLeftError(trialID, actorName)
		case waitForRecvEvent:
			return true, nil
		case started:
			return false, nil
		default:
			return false, ErrUnexpected
		}
	}
	_, err := manager.waitForTrial(ctx, trialID, actorName, isJoined)
	if err != nil {
		// TODO Should we un-start the trial in this case ?
		return err
	}
	return nil
}

// Act makes an actor act at a given tick and wait for the next received event.
//
// Act and RequestAct are used in conjunction: Act is used by the endpoint accessed by the actor code,
// RequestAct is used by the orchestrator's.
func (manager *Manager) Act(
	ctx context.Context,
	trialID string,
	actorName string,
	sentEvent SentEvent,
) (RecvEvent, error) {
	log := log.WithFields(logrus.Fields{"trial_id": trialID, "actor_name": actorName, "tick_id": sentEvent.TickID})

	trial := manager.getTrial(trialID, actorName)
	if trial == nil {
		return RecvEvent{}, NewTrialNotFoundError(trialID, actorName)
	}
	switch trial.Status() {
	case ended:
		log.Debug("Trial ended, ignore sent event, return last received event and removing it")
		lastRecvEvent := trial.LastRecvEvent()
		manager.removeTrial(trialID, actorName)
		return lastRecvEvent, nil
	case left:
		return RecvEvent{}, NewTrialLeftError(trialID, actorName)
	}

	log.Debug("Sending event")
	err := trial.SendEvent(sentEvent)
	if err != nil {
		return RecvEvent{}, err
	}
	manager.trialsObservable.Emit()

	log.Debug("Waiting for next received event")
	hasNextObservation := func(trial *actorTrial) (bool, error) {
		if trial == nil {
			return false, ErrUnexpected
		}
		switch trial.Status() {
		case ended:
			return true, nil
		case left:
			return false, NewTrialLeftError(trialID, actorName)
		case waitForSentEvent:
			return true, nil
		case waitForRecvEvent:
			return false, nil
		default:
			return false, ErrUnexpected
		}
	}
	trial, err = manager.waitForTrial(ctx, trialID, actorName, hasNextObservation)
	if err != nil {
		return RecvEvent{}, err
	}
	if trial.Status() == ended {
		manager.removeTrial(trialID, actorName)
	}
	return trial.LastRecvEvent(), nil
}

// RequestAct receives and event from the orchestrator and wait for an actor to send one.
//
// cf. Act
func (manager *Manager) RequestAct(
	ctx context.Context,
	trialID string,
	actorName string,
	receivedEvent RecvEvent,
) (SentEvent, error) {
	log := log.WithFields(logrus.Fields{"trial_id": trialID, "actor_name": actorName, "tick_id": receivedEvent.TickID})

	trial := manager.getTrial(trialID, actorName)
	if trial == nil {
		return SentEvent{}, NewTrialNotFoundError(trialID, actorName)
	}

	log.Debug("Receiving event")
	err := trial.ReceiveEvent(receivedEvent)
	if err != nil {
		return SentEvent{}, err
	}
	manager.trialsObservable.Emit()

	log.Debug("Waiting for next sent event")
	hasNextAction := func(trial *actorTrial) (bool, error) {
		if trial == nil {
			return false, ErrUnexpected
		}
		switch trial.Status() {
		case ended:
			return false, NewTrialEndedError(trialID)
		case left:
			return false, NewTrialLeftError(trialID, actorName)
		case waitForSentEvent:
			return false, nil
		case waitForRecvEvent:
			return true, nil
		default:
			return false, ErrUnexpected
		}
	}
	trial, err = manager.waitForTrial(ctx, trialID, actorName, hasNextAction)
	if err != nil {
		return SentEvent{}, err
	}
	return trial.LastSentEvent(), nil
}

// LeaveTrial explicitly ends the participation of an actor to a trial
func (manager *Manager) LeaveTrial(trialID string, actorName string) error {
	log := log.WithFields(logrus.Fields{"trial_id": trialID, "actor_name": actorName})
	log.Debug("Leaving trial")
	trial := manager.getTrial(trialID, actorName)
	if trial == nil {
		return NewTrialNotFoundError(trialID, actorName)
	}
	err := trial.Leave()
	if err != nil {
		return err
	}
	manager.trialsObservable.Emit()
	return nil
}

// EndTrial notify an actor that a trial has ended
func (manager *Manager) EndTrial(
	trialID string,
	actorName string,
	receivedEvent RecvEvent,
) error {
	log := log.WithFields(logrus.Fields{"trial_id": trialID, "actor_name": actorName})
	log.Debug("Ending the trial")
	trial := manager.getTrial(trialID, actorName)
	if trial == nil {
		return NewTrialNotFoundError(trialID, actorName)
	}
	err := trial.End(receivedEvent)
	if err != nil {
		return err
	}
	manager.trialsObservable.Emit()
	return nil
}
