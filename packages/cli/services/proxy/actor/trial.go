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
	"sync"

	"github.com/cogment/cogment/services/proxy/trialspec"
)

type actorTrialStatus int

const (
	joined           actorTrialStatus = iota // Trial has been joined but is not yet started
	started                                  // Trial has been started but is not yet joined
	waitForRecvEvent                         // Trial is started and the actor waits for the next received event
	waitForSentEvent                         // Trial is started and the actor waits for the next sent event
	left                                     // Trial has been left by the actor
	ended                                    // Trial has ended
)

func (status actorTrialStatus) String() string {
	return [...]string{
		"joined",
		"started",
		"waitForRecvEvent",
		"waitForSentEvent",
		"left",
		"ended",
	}[status]
}

// actorTrial represents a trial joined (or to be joined) by an actor
type actorTrial struct {
	lock sync.RWMutex

	trialID   string
	actorName string

	status actorTrialStatus

	actorClass          string
	actorImplementation string
	actorConfig         *trialspec.DynamicPbMessage

	lastRecvEvent RecvEvent
	lastSentEvent SentEvent
}

func newJoinedActorTrial(
	trialID string,
	actorName string,
) *actorTrial {
	return &actorTrial{
		lock:                sync.RWMutex{},
		trialID:             trialID,
		actorName:           actorName,
		status:              joined,
		actorClass:          "",
		actorImplementation: "",
		actorConfig:         nil,
		lastRecvEvent:       RecvEvent{},
		lastSentEvent:       SentEvent{},
	}
}

func newStartedActorTrial(
	trialID string,
	actorName string,
	actorClass string,
	actorImplementation string,
	actorConfig *trialspec.DynamicPbMessage,
) *actorTrial {
	return &actorTrial{
		lock:                sync.RWMutex{},
		trialID:             trialID,
		actorName:           actorName,
		status:              started,
		actorClass:          actorClass,
		actorImplementation: actorImplementation,
		actorConfig:         actorConfig,
		lastRecvEvent:       RecvEvent{},
		lastSentEvent:       SentEvent{},
	}
}

func (trial *actorTrial) Matches(trialID string, actorName string) bool {
	trial.lock.RLock()
	defer trial.lock.RUnlock()
	return trial.trialID == trialID && trial.actorName == actorName
}

type TrialInfo struct {
	TrialID             string `json:"trial_id,omitempty"`
	ActorName           string `json:"actor_name,omitempty"`
	ActorClass          string `json:"actor_class,omitempty"`
	ActorImplementation string `json:"actor_implementation,omitempty"`
}

func (trial *actorTrial) Info() TrialInfo {
	trial.lock.RLock()
	defer trial.lock.RUnlock()
	return TrialInfo{
		TrialID:             trial.trialID,
		ActorName:           trial.actorName,
		ActorClass:          trial.actorClass,
		ActorImplementation: trial.actorImplementation,
	}
}

func (trial *actorTrial) tickID() uint64 {
	tickID := trial.lastRecvEvent.TickID
	if trial.lastSentEvent.TickID > tickID {
		tickID = trial.lastSentEvent.TickID
	}
	return tickID
}

func (trial *actorTrial) Status() actorTrialStatus {
	trial.lock.RLock()
	defer trial.lock.RUnlock()
	return trial.status
}

func (trial *actorTrial) Start(
	actorClass string,
	actorImplementation string,
	actorConfig *trialspec.DynamicPbMessage,
) error {
	trial.lock.Lock()
	defer trial.lock.Unlock()
	if trial.status != joined {
		return NewTrialTransitionError(
			trial.trialID,
			trial.actorName,
			trial.status,
			waitForRecvEvent,
		)
	}
	trial.status = waitForRecvEvent
	trial.actorClass = actorClass
	trial.actorImplementation = actorImplementation
	trial.actorConfig = actorConfig
	return nil
}

func (trial *actorTrial) Join() error {
	trial.lock.Lock()
	defer trial.lock.Unlock()
	if trial.status != started {
		return NewTrialTransitionError(
			trial.trialID,
			trial.actorName,
			trial.status,
			waitForRecvEvent,
		)
	}
	trial.status = waitForRecvEvent
	return nil
}

func (trial *actorTrial) Leave() error {
	trial.lock.Lock()
	defer trial.lock.Unlock()
	if trial.status == left || trial.status == ended {
		return NewTrialTransitionError(
			trial.trialID,
			trial.actorName,
			trial.status,
			left,
		)
	}
	trial.status = left
	return nil
}

func (trial *actorTrial) SendEvent(event SentEvent) error {
	trial.lock.Lock()
	defer trial.lock.Unlock()
	if trial.status != waitForSentEvent ||
		trial.tickID() != event.TickID {
		return NewRunningTrialTransitionError(
			trial.trialID,
			trial.actorName,
			trial.status,
			trial.tickID(),
			waitForRecvEvent,
			event.TickID,
		)
	}
	trial.status = waitForRecvEvent
	trial.lastSentEvent = event
	return nil
}

func (trial *actorTrial) ReceiveEvent(event RecvEvent) error {
	trial.lock.Lock()
	defer trial.lock.Unlock()
	if trial.status != waitForRecvEvent ||
		trial.tickID() > event.TickID {
		return NewRunningTrialTransitionError(
			trial.trialID,
			trial.actorName,
			trial.status,
			trial.tickID(),
			waitForSentEvent,
			event.TickID,
		)
	}
	trial.status = waitForSentEvent
	trial.lastRecvEvent = event
	return nil
}

func (trial *actorTrial) End(event RecvEvent) error {
	trial.lock.Lock()
	defer trial.lock.Unlock()
	if trial.status == left ||
		trial.status == ended ||
		trial.tickID() > event.TickID {
		return NewRunningTrialTransitionError(
			trial.trialID,
			trial.actorName,
			trial.status,
			trial.tickID(),
			ended,
			event.TickID,
		)
	}
	trial.status = ended
	trial.lastRecvEvent = event
	return nil
}

func (trial *actorTrial) ActorConfig() *trialspec.DynamicPbMessage {
	trial.lock.RLock()
	defer trial.lock.RUnlock()
	return trial.actorConfig
}

func (trial *actorTrial) LastSentEvent() SentEvent {
	trial.lock.RLock()
	defer trial.lock.RUnlock()
	return trial.lastSentEvent
}

func (trial *actorTrial) LastRecvEvent() RecvEvent {
	trial.lock.RLock()
	defer trial.lock.RUnlock()
	return trial.lastRecvEvent
}
