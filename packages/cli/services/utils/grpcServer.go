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

package utils

import (
	"sync"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
)

func grpcCodeToLogrusLevel(code codes.Code) logrus.Level {
	switch code {
	case codes.OK:
		return logrus.DebugLevel
	case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.Unauthenticated:
		return logrus.InfoLevel
	case
		codes.DeadlineExceeded,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable:
		return logrus.WarnLevel
	case codes.Unimplemented, codes.Internal, codes.DataLoss, codes.Unknown:
		return logrus.ErrorLevel
	default:
		return logrus.ErrorLevel
	}
}

var internalGrpcLoggerSettingSingleton sync.Once

func NewGrpcServer(enableReflection bool, opts ...grpc.ServerOption) *grpc.Server {
	log := logrus.WithField("component", "grpc-server")
	internalGrpcLoggerSettingSingleton.Do(func() {
		// This should be done only once

		// Creating a logger dedicated to internal logging with the same settings as the default one
		internalLog := logrus.New()
		internalLog.SetOutput(logrus.StandardLogger().Out)
		internalLog.SetFormatter(logrus.StandardLogger().Formatter)

		// gRPC is quite verbose, only unleash it on debug or below
		internalLog.SetLevel(logrus.ErrorLevel)
		if logrus.StandardLogger().GetLevel() >= logrus.DebugLevel {
			internalLog.SetLevel(logrus.StandardLogger().GetLevel())
		}

		grpc_logrus.ReplaceGrpcLogger(internalLog.WithField("component", "grpc-server/internal"))
	})

	grpcLogrusOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(grpcCodeToLogrusLevel),
	}

	grpcServerOpts := append(opts,
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_logrus.UnaryServerInterceptor(logrus.WithField("component", "grpc-server/call"), grpcLogrusOpts...),
			grpc_prometheus.UnaryServerInterceptor,
		),
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_logrus.StreamServerInterceptor(logrus.WithField("component", "grpc-server/call"), grpcLogrusOpts...),
			grpc_prometheus.StreamServerInterceptor,
		),
	)
	server := grpc.NewServer(
		grpcServerOpts...,
	)

	if enableReflection {
		reflection.Register(server)
		log.Info("gRPC reflection server enabled")
	}

	return server
}
