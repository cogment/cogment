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

package grpcservers

import (
	"sync"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
)

func grpcCodeToLogrusLevel(code codes.Code) log.Level {
	switch code {
	case codes.OK:
		return log.DebugLevel
	case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.Unauthenticated:
		return log.InfoLevel
	case codes.DeadlineExceeded, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unavailable:
		return log.WarnLevel
	case codes.Unimplemented, codes.Internal, codes.DataLoss, codes.Unknown:
		return log.ErrorLevel
	default:
		return log.ErrorLevel
	}
}

func offsetLevel(level log.Level, offset int) log.Level {
	levelValue := int(level)
	offsetValue := levelValue + offset
	if offsetValue < int(log.PanicLevel) {
		return log.PanicLevel
	}
	if offsetValue > int(log.TraceLevel) {
		return log.TraceLevel
	}
	return log.Level(offsetValue)
}

var replaceInternalGrpcLoggerSingleton sync.Once

func CreateGrpcServer(enableReflection bool) *grpc.Server {
	globalLogLevel := log.GetLevel()

	replaceInternalGrpcLoggerSingleton.Do(func() {
		// Replacing the internal grpc logger
		grpcServerLog := log.New()
		grpcServerLog.SetLevel(offsetLevel(globalLogLevel, -1)) // This is really verbose so we set it one level above the global logger's
		grpcServerEntry := log.NewEntry(grpcServerLog)
		// This should be done only once before any call to `CreateGrpcServer`
		grpc_logrus.ReplaceGrpcLogger(grpcServerEntry)
	})

	// Logging calls
	grpcCallsLog := log.New()
	grpcCallsLog.SetLevel(globalLogLevel)
	grpcCallsEntry := log.NewEntry(grpcCallsLog)
	grpcLogrusOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(grpcCodeToLogrusLevel),
	}
	server := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_logrus.UnaryServerInterceptor(grpcCallsEntry, grpcLogrusOpts...),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_logrus.StreamServerInterceptor(grpcCallsEntry, grpcLogrusOpts...),
		),
	)

	if enableReflection {
		reflection.Register(server)
		log.Info("gRPC reflection server enabled")
	}

	return server
}
