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

package proxy

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // register in DefaultServerMux
	"time"

	"github.com/sirupsen/logrus"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/mwitkow/go-conntrack"
	"github.com/mwitkow/grpc-proxy/proxy"
	"golang.org/x/net/context" // register in DefaultServerMux
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var log = logrus.WithField("component", "orchestrator/grpcwebproxy")

var (
	maxCallRecvMsgSize  = 1024 * 1024 * 4  // Maximum receive message size limit - 4MB
	httpMaxWriteTimeout = 10 * time.Second // HTTP server config, max write duration.
	httpMaxReadTimeout  = 10 * time.Second // HTTP server config, max read duration.
	shutdownTimeout     = 10 * time.Second
)

type Options struct {
	BackendPort uint
	HTTPPort    uint
}

// This functions is the equivalent of https://github.com/improbable-eng/grpc-web/blob/v0.14.1/go/grpcwebproxy/main.go
// With the following options
// 	- backend_addr=options.BackendPort
// 	- run_tls_server=false
//	- allow_all_origins=true
//	- use_websockets=true
//	- server_http_debug_port=options.HTTPPort
func Run(ctx context.Context, options Options) error {
	backendHostPort := fmt.Sprintf("localhost:%d", options.BackendPort)
	grpcServer, err := buildGrpcProxyServer(backendHostPort)
	if err != nil {
		return err
	}

	wrappedGrpc := grpcweb.WrapServer(grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool { return true }),
	)

	httpServer := &http.Server{
		WriteTimeout: httpMaxWriteTimeout,
		ReadTimeout:  httpMaxReadTimeout,
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			wrappedGrpc.ServeHTTP(resp, req)
		}),
	}
	httpAddr := fmt.Sprintf("0.0.0.0:%d", options.HTTPPort)
	listener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		return fmt.Errorf("unable to listen on port %d: %w", options.HTTPPort, err)
	}
	httpListener := conntrack.NewListener(listener,
		conntrack.TrackWithName("http"),
		conntrack.TrackWithTcpKeepAlive(20*time.Second),
		conntrack.TrackWithTracing(),
	)

	log.WithField("address", httpListener.Addr().String()).Info("http grpcweb proxy listening")

	httpRes := make(chan error)
	go func() {
		if err := httpServer.Serve(httpListener); err != nil {
			httpRes <- fmt.Errorf("http server error: %w", err)
			return
		}
		httpRes <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, _ := context.WithTimeout(context.Background(), shutdownTimeout)
		err := httpServer.Shutdown(shutdownCtx)
		if err != nil {
			return fmt.Errorf("unable to shutdown the grpcwebproxy: %w", err)
		}
		err = <-httpRes
		if err != nil {
			return fmt.Errorf("error while waiting for the grpcwebproxy to shutdown: %w", err)
		}
		return ctx.Err()
	case err := <-httpRes:
		return err
	}
}

func buildGrpcProxyServer(backendHostPort string) (*grpc.Server, error) {
	// gRPC-wide changes.
	grpc.EnableTracing = true
	grpc_logrus.ReplaceGrpcLogger(log)

	// gRPC proxy logic.
	backendConn, err := dialBackendOrFail(backendHostPort)
	if err != nil {
		return nil, err
	}
	director := func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		outCtx, _ := context.WithCancel(ctx)
		mdCopy := md.Copy()
		delete(mdCopy, "user-agent")
		// If this header is present in the request from the web client,
		// the actual connection to the backend will not be established.
		// https://github.com/improbable-eng/grpc-web/issues/568
		delete(mdCopy, "connection")
		outCtx = metadata.NewOutgoingContext(outCtx, mdCopy)
		return outCtx, backendConn, nil
	}
	// Server with logging and monitoring enabled.
	return grpc.NewServer(
		grpc.CustomCodec(proxy.Codec()), //nolint
		grpc.UnknownServiceHandler(proxy.TransparentHandler(director)),
		grpc.MaxRecvMsgSize(maxCallRecvMsgSize),
		grpc_middleware.WithUnaryServerChain(
			grpc_logrus.UnaryServerInterceptor(log),
			grpc_prometheus.UnaryServerInterceptor,
		),
		grpc_middleware.WithStreamServerChain(
			grpc_logrus.StreamServerInterceptor(log),
			grpc_prometheus.StreamServerInterceptor,
		),
	), nil
}

func dialBackendOrFail(backendHostPort string) (*grpc.ClientConn, error) {
	opt := []grpc.DialOption{}
	opt = append(opt, grpc.WithCodec(proxy.Codec())) //nolint
	opt = append(opt, grpc.WithInsecure())

	opt = append(opt,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize)),
		grpc.WithBackoffMaxDelay(grpc.DefaultBackoffConfig.MaxDelay), //nolint
	)

	cc, err := grpc.Dial(backendHostPort, opt...)
	if err != nil {
		return nil, fmt.Errorf("failed dialing backend: %w", err)
	}
	return cc, nil
}
