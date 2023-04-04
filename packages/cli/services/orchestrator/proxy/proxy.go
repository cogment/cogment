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

package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof" // register in DefaultServerMux
	"time"

	"github.com/cogment/cogment/services/utils"
	"github.com/sirupsen/logrus"

	"github.com/cogment/cogment/version"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/mwitkow/go-conntrack"
	"github.com/mwitkow/grpc-proxy/proxy"
	"golang.org/x/net/context" // register in DefaultServerMux
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	WebPort     uint
}

// This functions is the equivalent of https://github.com/improbable-eng/grpc-web/blob/v0.14.1/go/grpcwebproxy/main.go
// With the following options
//   - backend_addr=options.BackendPort
//   - run_tls_server=false
//   - allow_all_origins=true
//   - use_websockets=true
//   - server_http_debug_port=options.WebPort
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
			isCorsPreflightRequest := wrappedGrpc.IsAcceptableGrpcCorsRequest(req)
			isGrpcWebRequest := wrappedGrpc.IsGrpcWebRequest(req)
			isGrpcWebSocketRequest := wrappedGrpc.IsGrpcWebSocketRequest(req)
			log.WithFields(logrus.Fields{
				"remote": req.RemoteAddr,
				"url":    req.URL,
				"method": req.Method,
			}).Trace("grpc-web endpoint HTTP request")

			if isCorsPreflightRequest || isGrpcWebRequest || isGrpcWebSocketRequest {
				if isCorsPreflightRequest {
					log.WithFields(logrus.Fields{
						"remote": req.RemoteAddr,
						"url":    req.URL,
					}).Debug("CORS preflight request")
				}
				wrappedGrpc.ServeHTTP(resp, req)
			} else {
				resp.Header().Set("Content-Type", "application/json")
				_, err := io.WriteString(resp, fmt.Sprintf("{"+
					"\"message\":\"This the grpc-web endpoint for Cogment Orchestrator\","+
					"\"version\":\"%s\","+
					"\"version_hash\":\"%s\""+
					"}", version.Version, version.Hash))
				if err != nil {
					log.Fatal("Error while writing response", err)
				}
			}
		}),
	}
	webAddr := fmt.Sprintf("0.0.0.0:%d", options.WebPort)
	listener, err := net.Listen("tcp", webAddr)
	if err != nil {
		return fmt.Errorf("unable to listen on port %d: %w", options.WebPort, err)
	}
	httpListener := conntrack.NewListener(listener,
		conntrack.TrackWithName("http"),
		conntrack.TrackWithTcpKeepAlive(20*time.Second),
		conntrack.TrackWithTracing(),
	)

	log.WithField("address", httpListener.Addr().String()).Info("server listening")

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
	return utils.NewGrpcServer(
		false,
		grpc.CustomCodec(proxy.Codec()), //nolint
		grpc.UnknownServiceHandler(proxy.TransparentHandler(director)),
		grpc.MaxRecvMsgSize(maxCallRecvMsgSize),
	), nil
}

func dialBackendOrFail(backendHostPort string) (*grpc.ClientConn, error) {
	opt := []grpc.DialOption{}
	opt = append(opt, grpc.WithCodec(proxy.Codec())) //nolint
	opt = append(opt, grpc.WithTransportCredentials(insecure.NewCredentials()))

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
