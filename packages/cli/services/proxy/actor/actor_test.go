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
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/stretchr/testify/assert"
)

const testDataDir string = "../../../testdata"

func generateName(t *testing.T, suffix string) string {
	return strings.ToLower(t.Name()) + suffix
}

func newManager(t *testing.T) *Manager {
	tsm, err := trialspec.NewFromFile(path.Join(testDataDir, "cogment.yaml"))
	if !assert.NoError(t, err) {
		assert.FailNow(t, "non-recoverable error")
	}
	manager, err := NewManager(tsm)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "non-recoverable error")
	}
	return manager
}

func TestNewManager(t *testing.T) {
	manager := newManager(t)

	t.Name()

	_, err := manager.GetTrial(generateName(t, "_trial"), generateName(t, "_actor_0"))
	assert.Error(t, err)
}

func TestJoinAndStartTrial(t *testing.T) {
	manager := newManager(t)
	defer manager.Destroy()

	wg := sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trialJoinedChan := make(chan struct{})

	// Start a worker that joins the trial
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := manager.JoinTrial(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
		)
		assert.NoError(t, err)
		assert.Equal(t, generateName(t, "_trial"), res.TrialID)
		assert.Equal(t, generateName(t, "_actor_0"), res.ActorName)
		assert.Equal(t, "ai_drone", res.ActorClass)
		assert.Equal(t, "actor_implementation_0", res.ActorImplementation)
		assert.Equal(t, uint64(0), res.TickID)
		assert.NotNil(t, res.ActorConfig)
		assert.NotNil(t, res.Observation)

		trialJoinedChan <- struct{}{}
	}()

	<-time.After(200 * time.Millisecond)

	// Later, start a worker that starts the trial and request the first action
	wg.Add(1)
	go func() {
		defer wg.Done()
		actorConfig, err := manager.Spec().NewActorConfig("ai_drone")
		assert.NoError(t, err)

		err = manager.StartTrial(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
			"ai_drone",
			"actor_implementation_0",
			actorConfig,
		)
		assert.NoError(t, err)

		observation, err := manager.Spec().NewObservation("ai_drone")
		assert.NoError(t, err)

		_, err = manager.RequestAct(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
			RecvEvent{
				TickID:      0,
				Observation: observation,
			},
		)

		// `RequestAct` should be canceled when the context of the test is canceled
		assert.ErrorIs(t, err, context.Canceled)
	}()

	// Wait for the trial to be joined
	<-trialJoinedChan
}

func TestStartAndJoinTrial(t *testing.T) {
	manager := newManager(t)
	defer manager.Destroy()

	wg := sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trialStartedChan := make(chan struct{})
	trialJoinedChan := make(chan struct{})
	trialLeftChan := make(chan struct{})

	// Start a worker that starts the trial and request the first action
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := manager.StartTrial(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
			"plane",
			"actor_implementation_0",
			nil,
		)
		assert.NoError(t, err)
		go func() {
			trialStartedChan <- struct{}{}
		}()

		observation, err := manager.Spec().NewObservation("plane")
		assert.NoError(t, err)

		_, err = manager.RequestAct(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
			RecvEvent{
				TickID:      0,
				Observation: observation,
			},
		)

		// `RequestAct` should be canceled when the actor leaves the trial later on
		var expectedErr *TrialLeftError
		assert.ErrorAs(t, err, &expectedErr)
		assert.Equal(t, generateName(t, "_trial"), expectedErr.TrialID)
		assert.Equal(t, generateName(t, "_actor_0"), expectedErr.ActorName)

		go func() {
			trialLeftChan <- struct{}{}
		}()
	}()

	<-time.After(200 * time.Millisecond)

	// Later, start a worker that joins the trial
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := manager.JoinTrial(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
		)
		assert.NoError(t, err)
		assert.Equal(t, generateName(t, "_trial"), res.TrialID)
		assert.Equal(t, generateName(t, "_actor_0"), res.ActorName)
		assert.Equal(t, "plane", res.ActorClass)
		assert.Equal(t, "actor_implementation_0", res.ActorImplementation)
		assert.Equal(t, uint64(0), res.TickID)
		assert.Nil(t, res.ActorConfig)
		assert.NotNil(t, res.Observation)

		go func() {
			trialJoinedChan <- struct{}{}
		}()
	}()

	<-trialStartedChan

	trialInfo, err := manager.GetTrial(generateName(t, "_trial"), generateName(t, "_actor_0"))
	assert.NoError(t, err)
	assert.Equal(t, generateName(t, "_trial"), trialInfo.TrialID)
	assert.Equal(t, generateName(t, "_actor_0"), trialInfo.ActorName)
	assert.Equal(t, "plane", trialInfo.ActorClass)
	assert.Equal(t, "actor_implementation_0", trialInfo.ActorImplementation)

	_, err = manager.GetTrial(generateName(t, "_trial"), "actor_1")
	var expectedErr *TrialNotFoundError
	assert.ErrorAs(t, err, &expectedErr)
	assert.Equal(t, generateName(t, "_trial"), expectedErr.TrialID)
	assert.Equal(t, "actor_1", expectedErr.ActorName)

	<-trialJoinedChan

	err = manager.LeaveTrial(generateName(t, "_trial"), generateName(t, "_actor_0"))
	assert.NoError(t, err)

	<-trialLeftChan
}

func TestStartAndJoinTrialTwice(t *testing.T) {
	manager := newManager(t)
	defer manager.Destroy()

	wg := sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a worker that starts the trial and request the first action
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := manager.StartTrial(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
			"plane",
			"actor_implementation_0",
			nil,
		)
		assert.NoError(t, err)

		observation, err := manager.Spec().NewObservation("plane")
		assert.NoError(t, err)

		_, err = manager.RequestAct(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
			RecvEvent{
				TickID:      0,
				Observation: observation,
			},
		)
		// `RequestAct` should be canceled when the context of the test is canceled
		assert.ErrorIs(t, err, context.Canceled)
	}()

	<-time.After(200 * time.Millisecond)

	// Later, start two worker that joins the trial
	joindWg := sync.WaitGroup{}
	for i := 0; i < 2; i++ {
		joindWg.Add(1)
		go func() {
			defer joindWg.Done()
			res, err := manager.JoinTrial(
				ctx,
				generateName(t, "_trial"),
				generateName(t, "_actor_0"),
			)
			assert.NoError(t, err)
			assert.Nil(t, res.ActorConfig)
			assert.NotNil(t, res.Observation)
		}()
	}

	joindWg.Wait()

	_, err := manager.GetTrial(generateName(t, "_trial"), generateName(t, "_actor_0"))
	assert.NoError(t, err)
}
func TestFullTrial(t *testing.T) {
	manager := newManager(t)
	defer manager.Destroy()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	defer wg.Wait()

	nbTicks := uint64(5)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := manager.StartTrial(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
			"plane",
			"actor_implementation_0",
			nil,
		)
		if !assert.NoError(t, err) {
			assert.FailNow(t, "non-recoverable error")
		}

		for tickID := uint64(0); tickID < nbTicks; tickID++ {
			observation, err := manager.Spec().NewObservation("plane")
			assert.NoError(t, err)

			recvEvent := RecvEvent{
				TickID:      tickID,
				Observation: observation,
			}

			if tickID > 0 {
				rewardUserData, err := manager.Spec().NewMessage("flames.ObjectInfo")
				assert.NoError(t, err)
				recvEvent.Rewards = []RecvReward{
					{
						TickID: tickID - 1,
						Value:  float32(tickID / nbTicks),
						Sources: []RecvRewardSource{
							{
								Sender:   "environment",
								Value:    float32(tickID / nbTicks),
								UserData: rewardUserData,
							},
						},
					},
				}
			}
			sentEvent, err := manager.RequestAct(
				ctx,
				generateName(t, "_trial"),
				generateName(t, "_actor_0"),
				recvEvent,
			)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "non-recoverable error")
			}
			assert.NotNil(t, sentEvent.Action)
			if tickID%2 == 0 {
				assert.Len(t, sentEvent.Rewards, 1)
				assert.Equal(t, generateName(t, "_actor_1"), sentEvent.Rewards[0].Receiver)
				assert.Equal(t, float32(0.5), sentEvent.Rewards[0].Confidence)
				assert.Equal(t, float32(1), sentEvent.Rewards[0].Value)
				assert.Equal(t, "bar", sentEvent.Rewards[0].UserData["foo"])
			} else {
				assert.Len(t, sentEvent.Rewards, 0)
			}
		}

		recvEvent := RecvEvent{
			TickID: nbTicks,
			Rewards: []RecvReward{
				{
					TickID: nbTicks - 1,
					Value:  12.0,
					Sources: []RecvRewardSource{
						{
							Sender:     "environment",
							Value:      12.0,
							Confidence: 1.0,
						},
					},
				},
			},
		}
		err = manager.EndTrial(
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
			recvEvent,
		)
		assert.NoError(t, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		joinRes, err := manager.JoinTrial(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
		)
		if !assert.NoError(t, err) {
			assert.FailNow(t, "non-recoverable error")
		}

		assert.Equal(t, generateName(t, "_trial"), joinRes.TrialID)
		assert.Equal(t, generateName(t, "_actor_0"), joinRes.ActorName)
		assert.Equal(t, "plane", joinRes.ActorClass)
		assert.Equal(t, "actor_implementation_0", joinRes.ActorImplementation)
		assert.Nil(t, joinRes.ActorConfig)

		tickID := joinRes.TickID
		assert.Equal(t, uint64(0), tickID)

		assert.NotNil(t, joinRes.Observation)

		for {
			action, err := manager.Spec().NewAction("plane")
			assert.NoError(t, err)

			sentEvent := SentEvent{
				TickID: tickID,
				Action: action,
			}

			if tickID%2 == 0 {
				sentEvent.Rewards = []SentReward{
					{
						TickID:     -1,
						Receiver:   generateName(t, "_actor_1"),
						Value:      1.,
						Confidence: 0.5,
						UserData: map[string]interface{}{
							"foo": "bar",
						},
					},
				}
			}

			recvEvent, err := manager.Act(
				ctx,
				generateName(t, "_trial"),
				generateName(t, "_actor_0"),
				sentEvent,
			)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "non-recoverable error")
			}
			assert.Greater(t, recvEvent.TickID, tickID)
			tickID = recvEvent.TickID

			if tickID == 0 {
				assert.NotNil(t, recvEvent.Observation)
				assert.Len(t, recvEvent.Rewards, 0)
			} else if tickID < nbTicks {
				assert.NotNil(t, recvEvent.Observation)

				assert.Len(t, recvEvent.Rewards, 1)
				assert.Equal(t, int(tickID-1), int(recvEvent.Rewards[0].TickID))
				assert.Equal(t, float32(tickID/nbTicks), recvEvent.Rewards[0].Value)

				assert.Len(t, recvEvent.Rewards[0].Sources, 1)
				assert.Equal(t, "environment", recvEvent.Rewards[0].Sources[0].Sender)
				assert.Equal(t, float32(tickID/nbTicks), recvEvent.Rewards[0].Sources[0].Value)
				assert.NotNil(t, recvEvent.Rewards[0].Sources[0].UserData)
			} else {
				assert.Nil(t, recvEvent.Observation)

				assert.Len(t, recvEvent.Rewards, 1)
				assert.Equal(t, int(tickID-1), int(recvEvent.Rewards[0].TickID))
				assert.Equal(t, float32(12), recvEvent.Rewards[0].Value)

				assert.Len(t, recvEvent.Rewards[0].Sources, 1)
				assert.Equal(t, "environment", recvEvent.Rewards[0].Sources[0].Sender)
				assert.Equal(t, float32(12), recvEvent.Rewards[0].Sources[0].Value)
				assert.Nil(t, recvEvent.Rewards[0].Sources[0].UserData)
				break
			}
		}
		_, err = manager.GetTrial(generateName(t, "_trial"), generateName(t, "_actor_0"))
		assert.Error(t, err)

		var expectedErr *TrialNotFoundError
		assert.ErrorAs(t, err, &expectedErr)
		assert.Equal(t, generateName(t, "_trial"), expectedErr.TrialID)
		assert.Equal(t, generateName(t, "_actor_0"), expectedErr.ActorName)
	}()
}

func TestFullTrialOutOfOrderAct(t *testing.T) {
	manager := newManager(t)
	defer manager.Destroy()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	defer wg.Wait()

	nbTicks := uint64(30)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := manager.StartTrial(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
			"plane",
			"actor_implementation_0",
			nil,
		)
		assert.NoError(t, err)

		for tickID := uint64(0); tickID < nbTicks; tickID++ {
			observation, err := manager.Spec().NewObservation("plane")
			assert.NoError(t, err)

			recvEvent := RecvEvent{
				TickID:      tickID,
				Observation: observation,
			}

			if tickID > 0 {
				rewardUserData, err := manager.Spec().NewMessage("flames.ObjectInfo")
				assert.NoError(t, err)
				recvEvent.Rewards = []RecvReward{
					{
						TickID: tickID - 1,
						Value:  float32(tickID / nbTicks),
						Sources: []RecvRewardSource{
							{
								Sender:   "environment",
								Value:    float32(tickID / nbTicks),
								UserData: rewardUserData,
							},
						},
					},
				}
			}
			_, err = manager.RequestAct(
				ctx,
				generateName(t, "_trial"),
				generateName(t, "_actor_0"),
				recvEvent,
			)
			assert.NoError(t, err)
		}

		err = manager.EndTrial(
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
			RecvEvent{
				TickID: nbTicks,
			},
		)
		assert.NoError(t, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		joinRes, err := manager.JoinTrial(
			ctx,
			generateName(t, "_trial"),
			generateName(t, "_actor_0"),
		)
		assert.NoError(t, err)

		tickID := joinRes.TickID

		for {
			action, err := manager.Spec().NewAction("plane")
			assert.NoError(t, err)

			if tickID == 12 {
				// `Act` call to a previous tick
				_, err := manager.Act(
					ctx,
					generateName(t, "_trial"),
					generateName(t, "_actor_0"),
					SentEvent{
						TickID: 11,
						Action: action,
					},
				)
				assert.Error(t, err)
			}
			recvEvent, err := manager.Act(
				ctx,
				generateName(t, "_trial"),
				generateName(t, "_actor_0"),
				SentEvent{
					TickID: tickID,
					Action: action,
				},
			)
			assert.NoError(t, err)
			tickID = recvEvent.TickID

			if recvEvent.Observation == nil {
				break
			}
		}
	}()
}
