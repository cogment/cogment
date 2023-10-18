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

package backend

import (
	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils"
)

// TrialSampleFilter represents the arguments to filter requested trial samples
type TrialSampleFilter struct {
	TrialIDs             []string
	ActorNames           []string
	ActorClasses         []string
	ActorImplementations []string
	Fields               []cogmentAPI.StoredTrialSampleField
}

// AppliedTrialSampleFilter represents a TrialSampleFilter applied to a particular trial
type AppliedTrialSampleFilter struct {
	trialParams  *cogmentAPI.TrialParams
	actorsFilter *arrayFilter
	fieldsFilter *arrayFilter
}

func newActorsFilter(filter TrialSampleFilter, trialParams *cogmentAPI.TrialParams) *arrayFilter {

	actorNamesFilter := utils.NewIDFilter(filter.ActorNames)
	actorClassesFilter := utils.NewIDFilter(filter.ActorClasses)
	actorImplsFilter := utils.NewIDFilter(filter.ActorImplementations)

	actorsFilter := newArrayFilter(len(trialParams.Actors))

	if actorNamesFilter.SelectsAll() && actorClassesFilter.SelectsAll() && actorImplsFilter.SelectsAll() {
		actorsFilter.setAll(true)
		return actorsFilter
	}

	actorsFilter.setAll(false)
	for actorIndex, actorParams := range trialParams.Actors {
		selectActorName := actorNamesFilter.SelectsAll()
		if !selectActorName {
			selectActorName = actorNamesFilter.Selects(actorParams.Name)
		}
		selectActorClass := actorClassesFilter.SelectsAll()
		if !selectActorClass {
			selectActorClass = actorClassesFilter.Selects(actorParams.ActorClass)
		}
		selectActorImpl := actorImplsFilter.SelectsAll()
		if !selectActorImpl {
			selectActorImpl = actorImplsFilter.Selects(actorParams.Implementation)
		}

		actorsFilter.set(actorIndex, selectActorName && selectActorClass && selectActorImpl)
	}
	return actorsFilter
}

func newFieldsFilter(fields []cogmentAPI.StoredTrialSampleField) *arrayFilter {
	fieldsFilter := newArrayFilter(len(cogmentAPI.StoredTrialSampleField_value))

	if len(fields) == 0 {
		fieldsFilter.setAll(true)
		return fieldsFilter
	}
	fieldsFilter.setAll(false)
	for _, field := range fields {
		if field != cogmentAPI.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_UNKNOWN {
			fieldsFilter.set(int(field), true)
		}
	}
	return fieldsFilter
}

func NewAppliedTrialSampleFilter(filter TrialSampleFilter, trialParams *cogmentAPI.TrialParams,
) *AppliedTrialSampleFilter {
	return &AppliedTrialSampleFilter{
		trialParams:  trialParams,
		actorsFilter: newActorsFilter(filter, trialParams),
		fieldsFilter: newFieldsFilter(filter.Fields),
	}
}

func (filter *AppliedTrialSampleFilter) SelectsAll() bool {
	return filter.actorsFilter.selectsAll() && filter.fieldsFilter.selectsAll()
}

func (filter *AppliedTrialSampleFilter) Filter(sample *cogmentAPI.StoredTrialSample) *cogmentAPI.StoredTrialSample {
	if filter.actorsFilter.selectsAll() && filter.fieldsFilter.selectsAll() {
		return sample
	}

	// Initialzed filtered sample and copy the unfilterable fields
	filteredSample := cogmentAPI.StoredTrialSample{
		UserId:       sample.UserId,
		TrialId:      sample.TrialId,
		TickId:       sample.TickId,
		Timestamp:    sample.Timestamp,
		State:        sample.State,
		ActorSamples: make([]*cogmentAPI.StoredTrialActorSample, 0, len(sample.ActorSamples)),
		Payloads:     make([][]byte, len(sample.Payloads)),
	}

	// Copy selected fields of selected agents
	for _, actorSample := range sample.ActorSamples {
		if filter.actorsFilter.selects(int(actorSample.Actor)) {
			filteredActorSample := cogmentAPI.StoredTrialActorSample{
				Actor:            actorSample.Actor,
				ReceivedRewards:  make([]*cogmentAPI.StoredTrialActorSampleReward, 0, len(actorSample.ReceivedRewards)),
				SentRewards:      make([]*cogmentAPI.StoredTrialActorSampleReward, 0, len(actorSample.SentRewards)),
				ReceivedMessages: make([]*cogmentAPI.StoredTrialActorSampleMessage, 0, len(actorSample.ReceivedMessages)),
				SentMessages:     make([]*cogmentAPI.StoredTrialActorSampleMessage, 0, len(actorSample.SentMessages)),
			}
			if actorSample.Observation != nil &&
				filter.fieldsFilter.selects(int(cogmentAPI.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_OBSERVATION)) {
				filteredActorSample.Observation = actorSample.Observation
				filteredSample.Payloads[*actorSample.Observation] = sample.Payloads[*actorSample.Observation]
			}

			if actorSample.Action != nil &&
				filter.fieldsFilter.selects(int(cogmentAPI.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_ACTION)) {
				filteredActorSample.Action = actorSample.Action
				filteredSample.Payloads[*actorSample.Action] = sample.Payloads[*actorSample.Action]
			}

			if filter.fieldsFilter.selects(int(cogmentAPI.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_REWARD)) {
				filteredActorSample.Reward = actorSample.Reward
			}

			if filter.fieldsFilter.selects(
				int(cogmentAPI.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_REWARDS)) {
				for _, reward := range actorSample.ReceivedRewards {
					filteredActorSample.ReceivedRewards = append(filteredActorSample.ReceivedRewards, reward)
					if reward.UserData != nil {
						filteredSample.Payloads[*reward.UserData] = sample.Payloads[*reward.UserData]
					}
				}
			}

			if filter.fieldsFilter.selects(int(cogmentAPI.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_REWARDS)) {
				for _, reward := range actorSample.SentRewards {
					filteredActorSample.SentRewards = append(filteredActorSample.SentRewards, reward)
					if reward.UserData != nil {
						filteredSample.Payloads[*reward.UserData] = sample.Payloads[*reward.UserData]
					}
				}
			}

			if filter.fieldsFilter.selects(
				int(cogmentAPI.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_MESSAGES)) {
				for _, message := range actorSample.ReceivedMessages {
					filteredActorSample.ReceivedMessages = append(filteredActorSample.ReceivedMessages, message)
					filteredSample.Payloads[message.Payload] = sample.Payloads[message.Payload]
				}
			}

			if filter.fieldsFilter.selects(int(cogmentAPI.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_MESSAGES)) {
				for _, message := range actorSample.SentMessages {
					filteredActorSample.SentMessages = append(filteredActorSample.SentMessages, message)
					filteredSample.Payloads[message.Payload] = sample.Payloads[message.Payload]
				}
			}
			filteredSample.ActorSamples = append(filteredSample.ActorSamples, &filteredActorSample)
		}
	}

	return &filteredSample
}

type arrayFilter []bool

func newArrayFilter(size int) *arrayFilter {
	filter := make(arrayFilter, size)
	for index := range filter {
		filter[index] = false
	}
	return &filter
}

func (filter *arrayFilter) setAll(selects bool) {
	for index := range *filter {
		(*filter)[index] = selects
	}
}

func (filter *arrayFilter) set(index int, selects bool) {
	(*filter)[index] = selects
}

func (filter *arrayFilter) selectsAll() bool {
	for index := range *filter {
		if !(*filter)[index] {
			return false
		}
	}
	return true
}

func (filter *arrayFilter) selects(index int) bool {
	return (*filter)[index]
}
