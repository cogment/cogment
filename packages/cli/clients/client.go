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

package clients

import (
	"context"
	"fmt"
	"net"

	"github.com/cogment/cogment/utils/endpoint"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	endpoint *endpoint.Endpoint
	dialer   func(context.Context, string) (net.Conn, error)
}

func CreateClientWithInsecureEndpoint(
	endpoint endpoint.Endpoint,
) (*Client, error) {
	return &Client{
		endpoint: &endpoint,
	}, nil
}

func CreateClientWithCustomDialer(dialer func(context.Context, string) (net.Conn, error)) *Client {
	return &Client{
		endpoint: nil,
		dialer:   dialer,
	}
}

func (c *Client) String() string {
	if c.dialer != nil {
		return "custom dialer"
	}
	return c.endpoint.String()
}

func (c *Client) Connect(ctx context.Context) (*grpc.ClientConn, error) {
	hasCustomDialer := c.dialer != nil
	hasEndpoint := c.endpoint != nil
	if !hasCustomDialer && !hasEndpoint {
		return nil, fmt.Errorf("unable to create connection, missing endpoint or dialer")
	}

	var opts []grpc.DialOption

	// At the moment we only support unencrypted
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// We want to wait until the connection is dialed up
	opts = append(opts, grpc.WithBlock())

	if hasEndpoint {
		url, err := c.endpoint.ResolvedURL()
		if err != nil {
			return nil, fmt.Errorf("unable to create connection, provided endpoint is not resolved (%w)", err)
		}
		connection, err := grpc.DialContext(ctx, url.Host, opts...)
		if err != nil {
			return nil, err
		}
		return connection, nil
	}

	// has custom dialer
	opts = append(opts, grpc.WithContextDialer(c.dialer))

	return grpc.DialContext(ctx, "custom_dialer", opts...)
}
