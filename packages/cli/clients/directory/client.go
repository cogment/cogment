// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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
	"net"
	"net/url"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	host   string
	port   string
	ctx    context.Context
	dialer func(context.Context, string) (net.Conn, error)
}

const directoryAuthTokenMetadataKey = "authentication-token"

func CreateClient(ctx context.Context, endpoint string, authenticationToken string) (*Client, error) {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("Not a valid URL [%s]: %w", endpoint, err)
	}

	if endpointURL.Scheme != "grpc" ||
		endpointURL.Path != "" ||
		endpointURL.RawQuery != "" ||
		endpointURL.RawFragment != "" ||
		endpointURL.User != nil {
		return nil, fmt.Errorf("Invalid grpc endpoint [%s] (expected 'grpc://<host>:<port>')", endpointURL)
	}

	client := &Client{}
	client.host = endpointURL.Hostname()
	client.port = endpointURL.Port()

	if authenticationToken != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, directoryAuthTokenMetadataKey, authenticationToken)
	}

	client.ctx = ctx

	return client, nil
}

func (client *Client) connect() (*grpc.ClientConn, error) {
	hasCustomDialer := client.dialer != nil
	hasInsecureEndpoint := client.host != "" && client.port != ""
	if !hasCustomDialer && !hasInsecureEndpoint {
		return nil, fmt.Errorf("Unable to create client connection, missing endpoint or dialer")
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	opts = append(opts, grpc.WithBlock())

	if hasInsecureEndpoint {
		address := client.host + ":" + client.port
		return grpc.DialContext(client.ctx, address, opts...)
	}

	opts = append(opts, grpc.WithContextDialer(client.dialer))
	return grpc.DialContext(client.ctx, "custom_dialer", opts...)
}

func (client *Client) Register(request *cogmentAPI.RegisterRequest) (uint64, string, error) {
	connection, err := client.connect()
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
	connection, err := client.connect()
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

func (client *Client) Inquire(request *cogmentAPI.InquireRequest,
) (*[]*cogmentAPI.FullServiceData, error) {
	connection, err := client.connect()
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
