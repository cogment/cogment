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
	grpcapi "github.com/cogment/cogment-trial-datastore/grpcapi/cogment/api"
)

type filter map[string]struct{}

func createFilterFromStringArray(array []string) filter {
	set := make(filter)
	for _, value := range array {
		set[value] = struct{}{}
	}
	return set
}

func (f *filter) selectsAll() bool {
	return len(*f) == 0
}

func (f *filter) selects(value string) bool {
	if len(*f) == 0 {
		return true
	}
	_, isSelected := (*f)[value]
	return isSelected
}

type trialActorFilter map[int]struct{}

func createActorFilter(actorNamesFilter filter, actorClassesFilter filter, actorImplsFilter filter, trialParams *grpcapi.TrialParams) trialActorFilter {
	selectedActors := make(trialActorFilter)
	if actorNamesFilter.selectsAll() && actorClassesFilter.selectsAll() && actorImplsFilter.selectsAll() {
		return selectedActors
	}
	selectAllActors := true
	for actorIdx, actorParams := range trialParams.Actors {
		selectActorName := actorNamesFilter.selectsAll()
		if !selectActorName {
			selectActorName = actorNamesFilter.selects(actorParams.Name)
		}
		selectActorClass := actorClassesFilter.selectsAll()
		if !selectActorName {
			selectActorClass = actorNamesFilter.selects(actorParams.ActorClass)
		}
		selectActorImpl := actorImplsFilter.selectsAll()
		if !selectActorImpl {
			selectActorImpl = actorNamesFilter.selects(actorParams.Implementation)
		}

		if selectActorName && selectActorClass && selectActorImpl {
			selectedActors[actorIdx] = struct{}{}
		} else {
			selectAllActors = false
		}
	}
	if selectAllActors {
		return make(trialActorFilter)
	}
	return selectedActors
}

func (f *trialActorFilter) selectsAll() bool {
	return len(*f) == 0
}

func (f *trialActorFilter) selects(actorIdx int) bool {
	if len(*f) == 0 {
		return true
	}
	_, isSelected := (*f)[actorIdx]
	return isSelected
}

type sampleFieldsFilter map[grpcapi.TrialSampleField]struct{}

func createSampleFieldsFilter(array []grpcapi.TrialSampleField) sampleFieldsFilter {
	selectedFields := make(sampleFieldsFilter)
	for _, selectedField := range array {
		if selectedField != grpcapi.TrialSampleField_TRIAL_SAMPLE_FIELD_UNKNOWN {
			selectedFields[selectedField] = struct{}{}
		}
	}
	if len(selectedFields) == 0 || len(grpcapi.TrialSampleField_name)-1 == len(selectedFields) {
		return make(sampleFieldsFilter)
	}
	return selectedFields
}

func (f *sampleFieldsFilter) selectsAll() bool {
	return len(*f) == 0
}

func (f *sampleFieldsFilter) selects(field grpcapi.TrialSampleField) bool {
	if len(*f) == 0 {
		return true
	}
	_, isSelected := (*f)[field]
	return isSelected
}

func filterTrialSample(sample *grpcapi.TrialSample, actorsFilter trialActorFilter, fieldsFilter sampleFieldsFilter) *grpcapi.TrialSample {
	if actorsFilter.selectsAll() && fieldsFilter.selectsAll() {
		return sample
	}

	// Copy the base
	filteredSample := grpcapi.TrialSample{
		UserId:       sample.UserId,
		TrialId:      sample.TrialId,
		TickId:       sample.TickId,
		Timestamp:    sample.Timestamp,
		State:        sample.State,
		ActorSamples: make([]*grpcapi.TrialActorSample, 0, len(sample.ActorSamples)),
		Payloads:     make([][]byte, len(sample.Payloads)),
	}

	// Copy selected fields of selected agents
	for _, actorSample := range sample.ActorSamples {
		if actorsFilter.selects(int(actorSample.Actor)) {
			filteredActorSample := grpcapi.TrialActorSample{
				Actor:            actorSample.Actor,
				ReceivedRewards:  make([]*grpcapi.TrialActorSampleReward, 0, len(actorSample.ReceivedRewards)),
				SentRewards:      make([]*grpcapi.TrialActorSampleReward, 0, len(actorSample.SentRewards)),
				ReceivedMessages: make([]*grpcapi.TrialActorSampleMessage, 0, len(actorSample.ReceivedMessages)),
				SentMessages:     make([]*grpcapi.TrialActorSampleMessage, 0, len(actorSample.SentMessages)),
			}
			if actorSample.Observation != nil && fieldsFilter.selects(grpcapi.TrialSampleField_TRIAL_SAMPLE_FIELD_OBSERVATION) {
				filteredActorSample.Observation = actorSample.Observation
				filteredSample.Payloads[*actorSample.Observation] = sample.Payloads[*actorSample.Observation]
			}

			if actorSample.Action != nil && fieldsFilter.selects(grpcapi.TrialSampleField_TRIAL_SAMPLE_FIELD_ACTION) {
				filteredActorSample.Action = actorSample.Action
				filteredSample.Payloads[*actorSample.Action] = sample.Payloads[*actorSample.Action]
			}

			if fieldsFilter.selects(grpcapi.TrialSampleField_TRIAL_SAMPLE_FIELD_REWARD) {
				filteredActorSample.Reward = actorSample.Reward
			}

			if fieldsFilter.selects(grpcapi.TrialSampleField_TRIAL_SAMPLE_FIELD_RECEIVED_REWARDS) {
				for _, reward := range actorSample.ReceivedRewards {
					filteredActorSample.ReceivedRewards = append(filteredActorSample.ReceivedRewards, reward)
					if reward.UserData != nil {
						filteredSample.Payloads[*reward.UserData] = sample.Payloads[*reward.UserData]
					}
				}
			}

			if fieldsFilter.selects(grpcapi.TrialSampleField_TRIAL_SAMPLE_FIELD_SENT_REWARDS) {
				for _, reward := range actorSample.SentRewards {
					filteredActorSample.SentRewards = append(filteredActorSample.SentRewards, reward)
					if reward.UserData != nil {
						filteredSample.Payloads[*reward.UserData] = sample.Payloads[*reward.UserData]
					}
				}
			}

			if fieldsFilter.selects(grpcapi.TrialSampleField_TRIAL_SAMPLE_FIELD_RECEIVED_MESSAGES) {
				for _, message := range actorSample.ReceivedMessages {
					filteredActorSample.ReceivedMessages = append(filteredActorSample.ReceivedMessages, message)
					filteredSample.Payloads[message.Payload] = sample.Payloads[message.Payload]
				}
			}

			if fieldsFilter.selects(grpcapi.TrialSampleField_TRIAL_SAMPLE_FIELD_SENT_MESSAGES) {
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
