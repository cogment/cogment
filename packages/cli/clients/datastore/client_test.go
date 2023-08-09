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

package datastore

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"testing"

	"github.com/cogment/cogment/clients"
	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	service "github.com/cogment/cogment/services/datastore"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type fixture struct {
	ctx    context.Context
	client *Client
}

// Test only client option to have it connects through a memory buffer
func createClientWithBufferListener(bufferListener *bufconn.Listener) (*Client, error) {
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return bufferListener.Dial()
	}

	client := &Client{
		Client: *clients.CreateClientWithCustomDialer(bufDialer),
	}

	return client, nil
}

func createFixture() (*fixture, error) {
	bufferListener := bufconn.Listen(1024 * 1024)
	serviceOptions := service.DefaultOptions
	serviceOptions.CustomListener = bufferListener
	go func() {
		if err := service.Run(context.Background(), serviceOptions); err != nil {
			log.Fatalf("Trial datastore service exited with failure: %v", err)
		}
	}()

	client, err := createClientWithBufferListener(bufferListener)
	if err != nil {
		return nil, err
	}

	return &fixture{
		ctx:    context.Background(),
		client: client,
	}, nil
}

func (fxt *fixture) destroy() {
}

func createTrials(t *testing.T, fxt *fixture, prefix string, trialCount int, properties map[string]string) {
	connection, err := fxt.client.Connect(fxt.ctx)
	assert.NoError(t, err)
	defer connection.Close()

	spClient := grpcapi.NewTrialDatastoreSPClient(connection)

	for i := 0; i < trialCount; i++ {
		ctx := metadata.AppendToOutgoingContext(fxt.ctx, "trial-id", fmt.Sprintf("%s%d", prefix, i))
		_, err := spClient.AddTrial(ctx, &grpcapi.AddTrialRequest{
			UserId: "test",
			TrialParams: &grpcapi.TrialParams{
				MaxSteps:   uint32(10 * (i + 1)),
				Properties: properties,
			},
		})
		assert.NoError(t, err)
	}
}

func createTrialSamples(t *testing.T, fxt *fixture, trialID string, sampleCount int, endTrial bool) {
	connection, err := fxt.client.Connect(fxt.ctx)
	assert.NoError(t, err)
	defer connection.Close()

	spClient := grpcapi.NewTrialDatastoreSPClient(connection)

	for i := 0; i < sampleCount; i++ {
		ctx := metadata.AppendToOutgoingContext(fxt.ctx, "trial-id", trialID)
		stream, err := spClient.AddSample(ctx)
		assert.NoError(t, err)
		state := grpcapi.TrialState_RUNNING
		if endTrial && i == sampleCount-1 {
			state = grpcapi.TrialState_ENDED
		}

		err = stream.Send(&grpcapi.AddSampleRequest{
			TrialSample: &grpcapi.StoredTrialSample{TrialId: trialID, UserId: "my_user", State: state},
		})
		assert.NoError(t, err)

		_, err = stream.CloseAndRecv()
		assert.NoError(t, err)
	}
}

func TestListTrialsNoTrials(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	rep, err := fxt.client.ListTrials(fxt.ctx, 10, "", map[string]string{})
	assert.NoError(t, err)
	assert.Empty(t, rep.TrialInfos)
}

func TestListTrialsSimple(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, fxt, "trial", 10, map[string]string{"key": "value"})

	rep1, err := fxt.client.ListTrials(fxt.ctx, 5, "", map[string]string{})
	assert.NoError(t, err)
	assert.Len(t, rep1.TrialInfos, 5)

	for _, trialInfo := range rep1.TrialInfos {
		assert.Equal(t, "test", trialInfo.UserId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, trialInfo.LastState)
		assert.Equal(t, "value", trialInfo.Params.Properties["key"])
	}

	rep2, err := fxt.client.ListTrials(fxt.ctx, 5, rep1.NextTrialHandle, map[string]string{})
	assert.NoError(t, err)
	assert.Len(t, rep2.TrialInfos, 5)

	for _, trialInfo := range rep2.TrialInfos {
		assert.Equal(t, "test", trialInfo.UserId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, trialInfo.LastState)
		assert.Equal(t, "value", trialInfo.Params.Properties["key"])
	}
}

func TestListTrialsFilter(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, fxt, "trialA-", 10, map[string]string{"key": "value"})
	createTrials(t, fxt, "trialB-", 10, map[string]string{"key2": "value2"})
	createTrials(t, fxt, "trialC-", 10, map[string]string{"key": "value", "key2": "value2"})

	rep1, err := fxt.client.ListTrials(fxt.ctx, 100, "", map[string]string{"key": "value"})
	assert.NoError(t, err)
	assert.Len(t, rep1.TrialInfos, 20)

	for _, trialInfo := range rep1.TrialInfos {
		assert.Equal(t, "test", trialInfo.UserId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, trialInfo.LastState)
		assert.Equal(t, "value", trialInfo.Params.Properties["key"])
		if value, ok := trialInfo.Params.Properties["key2"]; ok {
			assert.Equal(t, "value2", value)
		}
	}

	rep2, err := fxt.client.ListTrials(fxt.ctx, 100, "", map[string]string{"key": "value", "key2": "value2"})
	assert.NoError(t, err)
	assert.Len(t, rep2.TrialInfos, 10)

	for _, trialInfo := range rep2.TrialInfos {
		assert.Equal(t, "test", trialInfo.UserId)
		assert.Equal(t, grpcapi.TrialState_UNKNOWN, trialInfo.LastState)
		assert.Equal(t, "value", trialInfo.Params.Properties["key"])
		assert.Equal(t, "value2", trialInfo.Params.Properties["key2"])
	}
}

func TestExportEmptyTrials(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, fxt, "trial", 10, map[string]string{"key": "value"})

	var buf bytes.Buffer
	_, err = fxt.client.ExportTrials(fxt.ctx, []string{"trial0", "trial2"}, &buf)
	assert.Error(t, err)
	assert.EqualError(t, err, "no samples to export for selected trials")
}

func TestExportImportTrialsSimple(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, fxt, "trial", 3, map[string]string{"key": "value"})

	createTrialSamples(t, fxt, "trial0", 20, true)
	createTrialSamples(t, fxt, "trial1", 10, false)
	createTrialSamples(t, fxt, "trial2", 30, false)

	var buf bytes.Buffer
	size, err := fxt.client.ExportTrials(fxt.ctx, []string{"trial0", "trial2"}, &buf)
	assert.NoError(t, err)
	assert.Equal(t, size, buf.Len())

	reader := bytes.NewReader(buf.Bytes())
	trialSamplesCount, err := fxt.client.ImportTrials(fxt.ctx, "import-test", "prefix-", reader)
	assert.NoError(t, err)
	assert.Len(t, trialSamplesCount, 2)
	assert.Equal(t, 20, trialSamplesCount["prefix-trial0"])
	assert.Equal(t, 30, trialSamplesCount["prefix-trial2"])

	rep, err := fxt.client.ListTrials(fxt.ctx, 10, "", map[string]string{})
	assert.NoError(t, err)
	assert.Len(t, rep.TrialInfos, 5)
	for _, trialInfo := range rep.TrialInfos {
		assert.Contains(t, []string{"trial0", "trial1", "trial2", "prefix-trial0", "prefix-trial2"}, trialInfo.TrialId)
		if trialInfo.TrialId == "trial0" {
			assert.Equal(t, 20, int(trialInfo.SamplesCount))
			assert.Equal(t, "test", trialInfo.UserId)
		} else if trialInfo.TrialId == "trial1" {
			assert.Equal(t, 10, int(trialInfo.SamplesCount))
			assert.Equal(t, "test", trialInfo.UserId)
		} else if trialInfo.TrialId == "prefix-trial0" {
			assert.Equal(t, 20, int(trialInfo.SamplesCount))
			assert.Equal(t, "import-test", trialInfo.UserId)
			assert.Equal(t, grpcapi.TrialState_ENDED, trialInfo.LastState)
		} else if trialInfo.TrialId == "prefix-trial2" {
			assert.Equal(t, 30, int(trialInfo.SamplesCount))
			assert.Equal(t, "import-test", trialInfo.UserId)
			assert.Equal(t, grpcapi.TrialState_RUNNING, trialInfo.LastState)
		}
	}
}

func TestExportImportLotsOfTrials(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	trialsCount := chunkTrialsCount*5 + 2 // Make sure trials will be requested in multiple chunks
	createTrials(t, fxt, "trial", trialsCount, map[string]string{"key": "value"})

	trialIDs := []string{}
	for trialIdx := 0; trialIdx < trialsCount; trialIdx++ {
		trialID := fmt.Sprintf("trial%d", trialIdx)
		createTrialSamples(t, fxt, trialID, 1000, true)
		trialIDs = append(trialIDs, trialID)
	}

	var buf bytes.Buffer
	size, err := fxt.client.ExportTrials(fxt.ctx, trialIDs, &buf)
	assert.NoError(t, err)
	assert.Equal(t, size, buf.Len())

	reader := bytes.NewReader(buf.Bytes())
	trialSamplesCount, err := fxt.client.ImportTrials(fxt.ctx, "import-test", "prefix-", reader)
	assert.NoError(t, err)
	assert.Len(t, trialSamplesCount, trialsCount)
}

func TestExportImportTrialsDuplicate(t *testing.T) {
	fxt, err := createFixture()
	assert.NoError(t, err)
	defer fxt.destroy()

	createTrials(t, fxt, "trial", 3, map[string]string{"key": "value"})

	createTrialSamples(t, fxt, "trial0", 20, true)
	createTrialSamples(t, fxt, "trial1", 10, true)
	createTrialSamples(t, fxt, "trial2", 15, true)

	var buf bytes.Buffer
	size, err := fxt.client.ExportTrials(fxt.ctx, []string{"trial0", "trial1", "trial2"}, &buf)
	assert.NoError(t, err)
	assert.Equal(t, size, buf.Len())

	err = fxt.client.DeleteTrials(fxt.ctx, []string{"trial1"})
	assert.NoError(t, err)

	reader := bytes.NewReader(buf.Bytes())
	_, err = fxt.client.ImportTrials(fxt.ctx, "import-test", "", reader)
	assert.Error(t, err)
	assert.EqualError(t, err, "trials [trial0 trial2] already exist in target datastore, import aborted")
}
