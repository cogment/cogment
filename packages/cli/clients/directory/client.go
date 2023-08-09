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

package directory

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
	ctx context.Context
}

const directoryAuthTokenMetadataKey = "authentication-token"

func CreateClient(ctx context.Context, endpoint endpoint.Endpoint, authenticationToken string) (*Client, error) {
	subClient, err := clients.CreateClientWithInsecureEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	client := &Client{Client: *subClient}

	if authenticationToken != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, directoryAuthTokenMetadataKey, authenticationToken)
	}

	client.ctx = ctx

	return client, nil
}

func (client *Client) Register(request *cogmentAPI.RegisterRequest) (uint64, string, error) {
	connection, err := client.Connect(client.ctx)
	if err != nil {
		return 0, "", err
	}
	defer connection.Close()

	grpcClient := cogmentAPI.NewDirectorySPClient(connection)
	clientStream, err := grpcClient.Register(client.ctx, grpc.WaitForReady(true))
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = clientStream.CloseSend() }()

	err = clientStream.Send(request)
	if err != nil {
		return 0, "", err
	}
	reply, err := clientStream.Recv()
	if err != nil {
		return 0, "", err
	}

	if reply.Status != cogmentAPI.RegisterReply_OK {
		return 0, "", fmt.Errorf("Service registration failed [%s]", reply.ErrorMsg)
	}

	return reply.ServiceId, reply.Secret, nil
}

func (client *Client) Deregister(request *cogmentAPI.DeregisterRequest) error {
	connection, err := client.Connect(client.ctx)
	if err != nil {
		return err
	}
	defer connection.Close()

	grpcClient := cogmentAPI.NewDirectorySPClient(connection)
	clientStream, err := grpcClient.Deregister(client.ctx, grpc.WaitForReady(true))
	if err != nil {
		return err
	}
	defer func() { _ = clientStream.CloseSend() }()

	err = clientStream.Send(request)
	if err != nil {
		return err
	}

	reply, err := clientStream.Recv()
	if err != nil {
		return err
	}

	if reply.Status != cogmentAPI.DeregisterReply_OK {
		return fmt.Errorf("Service deregistration failed [%s]", reply.ErrorMsg)
	}

	return nil
}

func (client *Client) Inquire(request *cogmentAPI.InquireRequest) (*[]*cogmentAPI.FullServiceData, error) {
	connection, err := client.Connect(client.ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()

	grpcClient := cogmentAPI.NewDirectorySPClient(connection)
	clientStream, err := grpcClient.Inquire(client.ctx, request, grpc.WaitForReady(true))
	if err != nil {
		return nil, err
	}

	var result []*cogmentAPI.FullServiceData
	for {
		reply, err := clientStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		result = append(result, reply.Data)
	}

	return &result, nil
}

func (client *Client) WaitForReady() error {
	request := cogmentAPI.InquireRequest{}
	request.Inquiry = &cogmentAPI.InquireRequest_ServiceId{ServiceId: 0}

	connection, err := client.Connect(client.ctx)
	if err != nil {
		return err
	}
	defer connection.Close()

	grpcClient := cogmentAPI.NewDirectorySPClient(connection)
	_, err = grpcClient.Inquire(client.ctx, &request, grpc.WaitForReady(true))
	if err != nil {
		return err
	}

	return nil
}
