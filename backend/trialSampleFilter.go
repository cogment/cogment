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
	"github.com/cogment/cogment-trial-datastore/utils"
)

// TrialSampleFilter represents the arguments to filter requested trial samples
type TrialSampleFilter struct {
	TrialIDs             []string
	ActorNames           []string
	ActorClasses         []string
	ActorImplementations []string
	Fields               []grpcapi.StoredTrialSampleField
}

// AppliedTrialSampleFilter represents a TrialSampleFilter applied to a particular trial
type AppliedTrialSampleFilter struct {
	trialParams  *grpcapi.TrialParams
	actorsFilter *idxFilter
	fieldsFilter *idxFilter
}

func newActorsFilter(filter TrialSampleFilter, trialParams *grpcapi.TrialParams) *idxFilter {

	actorNamesFilter := utils.NewIDFilter(filter.ActorNames)
	actorClassesFilter := utils.NewIDFilter(filter.ActorClasses)
	actorImplsFilter := utils.NewIDFilter(filter.ActorImplementations)

	if actorNamesFilter.SelectsAll() && actorClassesFilter.SelectsAll() && actorImplsFilter.SelectsAll() {
		return newIdxFilter([]int{})
	}

	actorsFilter := newIdxFilter([]int{})
	selectAllActors := true
	for actorIdx, actorParams := range trialParams.Actors {
		selectActorName := actorNamesFilter.SelectsAll()
		if !selectActorName {
			selectActorName = actorNamesFilter.Selects(actorParams.Name)
		}
		selectActorClass := actorClassesFilter.SelectsAll()
		if !selectActorName {
			selectActorClass = actorNamesFilter.Selects(actorParams.ActorClass)
		}
		selectActorImpl := actorImplsFilter.SelectsAll()
		if !selectActorImpl {
			selectActorImpl = actorNamesFilter.Selects(actorParams.Implementation)
		}

		if selectActorName && selectActorClass && selectActorImpl {
			actorsFilter.add(actorIdx)
		} else {
			selectAllActors = false
		}
	}
	if selectAllActors {
		return newIdxFilter([]int{})
	}

	return actorsFilter
}

func newFieldsFilter(fields []grpcapi.StoredTrialSampleField) *idxFilter {
	fieldsFilter := newIdxFilter([]int{})
	for _, field := range fields {
		if field != grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_UNKNOWN {
			fieldsFilter.add(int(field))
		}
	}
	if len(grpcapi.StoredTrialSampleField_name)-1 == len(*fieldsFilter) {
		return newIdxFilter([]int{})
	}
	return fieldsFilter
}

func NewAppliedTrialSampleFilter(filter TrialSampleFilter, trialParams *grpcapi.TrialParams) *AppliedTrialSampleFilter {
	return &AppliedTrialSampleFilter{
		trialParams:  trialParams,
		actorsFilter: newActorsFilter(filter, trialParams),
		fieldsFilter: newFieldsFilter(filter.Fields),
	}
}

func (f *AppliedTrialSampleFilter) SelectsAll() bool {
	return f.actorsFilter.selectsAll() && f.fieldsFilter.selectsAll()
}

func (f *AppliedTrialSampleFilter) Filter(sample *grpcapi.StoredTrialSample) *grpcapi.StoredTrialSample {
	if f.actorsFilter.selectsAll() && f.fieldsFilter.selectsAll() {
		return sample
	}

	// Copy the base
	filteredSample := grpcapi.StoredTrialSample{
		UserId:       sample.UserId,
		TrialId:      sample.TrialId,
		TickId:       sample.TickId,
		Timestamp:    sample.Timestamp,
		State:        sample.State,
		ActorSamples: make([]*grpcapi.StoredTrialActorSample, 0, len(sample.ActorSamples)),
		Payloads:     make([][]byte, len(sample.Payloads)),
	}

	// Copy selected fields of selected agents
	for _, actorSample := range sample.ActorSamples {
		if f.actorsFilter.selects(int(actorSample.Actor)) {
			filteredActorSample := grpcapi.StoredTrialActorSample{
				Actor:            actorSample.Actor,
				ReceivedRewards:  make([]*grpcapi.StoredTrialActorSampleReward, 0, len(actorSample.ReceivedRewards)),
				SentRewards:      make([]*grpcapi.StoredTrialActorSampleReward, 0, len(actorSample.SentRewards)),
				ReceivedMessages: make([]*grpcapi.StoredTrialActorSampleMessage, 0, len(actorSample.ReceivedMessages)),
				SentMessages:     make([]*grpcapi.StoredTrialActorSampleMessage, 0, len(actorSample.SentMessages)),
			}
			if actorSample.Observation != nil && f.fieldsFilter.selects(int(grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_OBSERVATION)) {
				filteredActorSample.Observation = actorSample.Observation
				filteredSample.Payloads[*actorSample.Observation] = sample.Payloads[*actorSample.Observation]
			}

			if actorSample.Action != nil && f.fieldsFilter.selects(int(grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_ACTION)) {
				filteredActorSample.Action = actorSample.Action
				filteredSample.Payloads[*actorSample.Action] = sample.Payloads[*actorSample.Action]
			}

			if f.fieldsFilter.selects(int(grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_REWARD)) {
				filteredActorSample.Reward = actorSample.Reward
			}

			if f.fieldsFilter.selects(int(grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_REWARDS)) {
				for _, reward := range actorSample.ReceivedRewards {
					filteredActorSample.ReceivedRewards = append(filteredActorSample.ReceivedRewards, reward)
					if reward.UserData != nil {
						filteredSample.Payloads[*reward.UserData] = sample.Payloads[*reward.UserData]
					}
				}
			}

			if f.fieldsFilter.selects(int(grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_REWARDS)) {
				for _, reward := range actorSample.SentRewards {
					filteredActorSample.SentRewards = append(filteredActorSample.SentRewards, reward)
					if reward.UserData != nil {
						filteredSample.Payloads[*reward.UserData] = sample.Payloads[*reward.UserData]
					}
				}
			}

			if f.fieldsFilter.selects(int(grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_RECEIVED_MESSAGES)) {
				for _, message := range actorSample.ReceivedMessages {
					filteredActorSample.ReceivedMessages = append(filteredActorSample.ReceivedMessages, message)
					filteredSample.Payloads[message.Payload] = sample.Payloads[message.Payload]
				}
			}

			if f.fieldsFilter.selects(int(grpcapi.StoredTrialSampleField_STORED_TRIAL_SAMPLE_FIELD_SENT_MESSAGES)) {
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

type idxFilter map[int]struct{}

func newIdxFilter(selectedIdxs []int) *idxFilter {
	f := make(idxFilter)
	for _, idx := range selectedIdxs {
		f[idx] = struct{}{}
	}
	return &f
}

func (f *idxFilter) add(idx int) {
	(*f)[idx] = struct{}{}
}

func (f *idxFilter) selectsAll() bool {
	return len(*f) == 0
}

func (f *idxFilter) selects(idx int) bool {
	if len(*f) == 0 {
		return true
	}
	_, isSelected := (*f)[idx]
	return isSelected
}
