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

package control

import (
	"context"
	"fmt"
	"io"

	"github.com/cogment/cogment/clients"
	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils/endpoint"
)

type Client struct {
	clients.Client
	userID string
}

func CreateClientWithInsecureEndpoint(endpoint endpoint.Endpoint, userID string) (*Client, error) {
	subClient, err := clients.CreateClientWithInsecureEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	client := &Client{
		Client: *subClient,
		userID: userID,
	}

	return client, nil
}

func (client *Client) WatchTrials(ctx context.Context, out chan<- *grpcapi.TrialInfo) error {
	connection, err := client.Connect(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()

	spClient := grpcapi.NewTrialLifecycleSPClient(connection)

	request := grpcapi.TrialListRequest{
		FullInfo: true,
	}
	stream, err := spClient.WatchTrials(ctx, &request)
	if err != nil {
		return err
	}
	for {
		entry, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		out <- entry.Info
	}
	return fmt.Errorf(
		"Unexpected end of the return stream for WatchTrials() call on [{%s}]",
		client,
	)
}

func (client *Client) StartTrial(
	ctx context.Context,
	requestedTrialID string,
	params *grpcapi.TrialParams,
) (string, error) {
	connection, err := client.Connect(ctx)
	if err != nil {
		return "", err
	}
	defer connection.Close()

	spClient := grpcapi.NewTrialLifecycleSPClient(connection)

	request := grpcapi.TrialStartRequest{
		UserId:           client.userID,
		TrialIdRequested: requestedTrialID,
		StartData: &grpcapi.TrialStartRequest_Params{
			Params: params,
		},
	}
	reply, err := spClient.StartTrial(ctx, &request)
	if err != nil {
		return "", err
	}
	return reply.TrialId, nil
}
