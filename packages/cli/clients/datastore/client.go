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
	"context"
	"fmt"
	"io"

	"github.com/cogment/cogment/clients"
	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils/endpoint"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	clients.Client
}

const chunkTrialsCount = 20

func CreateClientWithInsecureEndpoint(endpoint *endpoint.Endpoint) (*Client, error) {
	subClient, err := clients.CreateClientWithInsecureEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	client := &Client{Client: *subClient}

	return client, nil
}

func (client *Client) ListTrials(
	ctx context.Context,
	trialsCount uint,
	fromHandle string,
	properties map[string]string,
) (*cogmentAPI.RetrieveTrialsReply, error) {
	connection, err := client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()

	spClient := cogmentAPI.NewTrialDatastoreSPClient(connection)

	req := &cogmentAPI.RetrieveTrialsRequest{
		TrialsCount: uint32(trialsCount),
		TrialHandle: fromHandle,
		Properties:  properties,
	}

	rep, err := spClient.RetrieveTrials(ctx, req, grpc.WaitForReady(true))
	if err != nil {
		return nil, err
	}

	return rep, nil
}

func (client *Client) DeleteTrials(ctx context.Context, trialIDs []string) error {
	connection, err := client.Connect(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()

	spClient := cogmentAPI.NewTrialDatastoreSPClient(connection)

	req := &cogmentAPI.DeleteTrialsRequest{
		TrialIds: trialIDs,
	}

	_, err = spClient.DeleteTrials(ctx, req, grpc.WaitForReady(true))
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) ExportTrials(ctx context.Context, trialIDs []string, writer io.Writer) (int, error) {
	connection, err := client.Connect(ctx)
	if err != nil {
		return 0, err
	}
	defer connection.Close()

	spClient := cogmentAPI.NewTrialDatastoreSPClient(connection)

	// Request the trial params
	trialParams := make(map[string]*cogmentAPI.TrialParams)
	toBeWrittenSamples := make(map[string]uint32)
	for chunkStartIdx := 0; chunkStartIdx < len(trialIDs); chunkStartIdx += chunkTrialsCount {
		chunkEndIdx := chunkStartIdx + chunkTrialsCount
		if chunkEndIdx > len(trialIDs) {
			chunkEndIdx = len(trialIDs)
		}
		trialIDsChunk := trialIDs[chunkStartIdx:chunkEndIdx]

		retrieveTrialsReq := &cogmentAPI.RetrieveTrialsRequest{
			TrialIds: trialIDsChunk,
		}

		retrieveTrialsRep, err := spClient.RetrieveTrials(ctx, retrieveTrialsReq, grpc.WaitForReady(true))
		if err != nil {
			return 0, err
		}

		// Retrieve the trial params and the current sample count for each trials
		for _, trialInfo := range retrieveTrialsRep.TrialInfos {
			if trialInfo.SamplesCount > 0 {
				trialParams[trialInfo.TrialId] = trialInfo.Params
				toBeWrittenSamples[trialInfo.TrialId] = trialInfo.SamplesCount
			}
		}
	}

	if len(toBeWrittenSamples) == 0 {
		// No samples to be written
		return 0, fmt.Errorf("no samples to export for selected trials")
	}

	// Request the samples
	retrieveSamplesReq := &cogmentAPI.RetrieveSamplesRequest{
		TrialIds: trialIDs,
	}

	stream, err := spClient.RetrieveSamples(ctx, retrieveSamplesReq, grpc.WaitForReady(true))
	if err != nil {
		return 0, err
	}

	// Start writing the file
	fileWriter := CreateTrialSamplesFileWriter(writer)
	err = fileWriter.WriteHeader(trialParams)
	if err != nil {
		return fileWriter.Bytes, err
	}

	for {
		select {
		case <-ctx.Done():
			// The context has been cancelled
			return fileWriter.Bytes, ctx.Err()
		default:
			break
		}

		// Receive sample and write it to the file
		rep, err := stream.Recv()
		if err == io.EOF {
			// Reached the end
			return fileWriter.Bytes, nil
		}
		if err != nil {
			return fileWriter.Bytes, err
		}
		sample := rep.TrialSample
		if _, exists := toBeWrittenSamples[sample.TrialId]; !exists {
			// Already written the expected number of samples for this trial
			continue
		}
		err = fileWriter.WriteSample(rep.TrialSample)
		if err != nil {
			return fileWriter.Bytes, err
		}
		toBeWrittenSamples[sample.TrialId]--
		if toBeWrittenSamples[sample.TrialId] == 0 {
			delete(toBeWrittenSamples, sample.TrialId)
		}
		if len(toBeWrittenSamples) == 0 {
			// Reached the end of what we were meaning to write
			return fileWriter.Bytes, nil
		}
	}
}

func (client *Client) ImportTrials(
	ctx context.Context,
	userID string,
	trialIDPrefix string,
	reader io.Reader,
) (map[string]int, error) {
	connection, err := client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()

	spClient := cogmentAPI.NewTrialDatastoreSPClient(connection)
	fileReader := CreateTrialSamplesFileReader(reader)

	prefixedTrialIDs := []string{}

	// Read the header
	header, err := fileReader.ReadHeader()
	if err != nil {
		return nil, err
	}

	// Retrieve the list of trial ids
	for trialID := range header.TrialParams {
		prefixedTrialIDs = append(prefixedTrialIDs, trialIDPrefix+trialID)
	}

	// Check if some trials already exist
	req := &cogmentAPI.RetrieveTrialsRequest{
		TrialIds: prefixedTrialIDs,
	}
	rep, err := spClient.RetrieveTrials(ctx, req, grpc.WaitForReady(true))
	if err != nil {
		return nil, err
	}
	existingTrialIDs := []string{}
	for _, trialInfo := range rep.TrialInfos {
		for _, toImportTrialID := range prefixedTrialIDs {
			if trialInfo.TrialId == toImportTrialID {
				existingTrialIDs = append(existingTrialIDs, toImportTrialID)
			}
		}
	}
	if len(existingTrialIDs) > 0 {
		return nil, fmt.Errorf("trials %v already exist in target datastore, import aborted", existingTrialIDs)
	}

	// Read the samples
	trialSamplesCount := make(map[string]int)
	for {
		select {
		case <-ctx.Done():
			// The context has been cancelled
			return nil, ctx.Err()
		default:
			break
		}

		// Read sample
		sample, err := fileReader.ReadSample()
		if err == io.EOF {
			// Reached the end of the file
			return trialSamplesCount, nil
		}
		if err != nil {
			return nil, err
		}

		prefixedTrialID := trialIDPrefix + sample.TrialId
		if _, exists := trialSamplesCount[prefixedTrialID]; !exists {
			// First time we encounter this trial, adding it
			trialParams, exists := header.TrialParams[sample.TrialId]
			if !exists {
				return nil, fmt.Errorf(
					"read sample from trial [%s], this trial parameters are missing from the file",
					sample.TrialId,
				)
			}
			trialAddCtx := metadata.AppendToOutgoingContext(ctx, "trial-id", prefixedTrialID)
			req := &cogmentAPI.AddTrialRequest{
				UserId:      userID,
				TrialParams: trialParams,
			}
			_, err := spClient.AddTrial(trialAddCtx, req, grpc.WaitForReady(true))
			if err != nil {
				return nil, err
			}

			trialSamplesCount[prefixedTrialID] = 0
		}

		// Add the sample
		sampleAddCtx := metadata.AppendToOutgoingContext(ctx, "trial-id", prefixedTrialID)
		stream, err := spClient.AddSample(sampleAddCtx, grpc.WaitForReady(true))
		if err != nil {
			return nil, err
		}

		sample.TrialId = ""
		err = stream.Send(&cogmentAPI.AddSampleRequest{
			TrialSample: sample,
		})
		if err != nil {
			return nil, err
		}

		_, err = stream.CloseAndRecv()
		if err != nil {
			return nil, err
		}

		trialSamplesCount[prefixedTrialID]++
	}
}
