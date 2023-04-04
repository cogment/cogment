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

package grpcservers

import (
	"context"

	"github.com/cogment/cogment/services/modelRegistry/backend"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BackendPromise struct {
	backend backend.Backend
	updated chan struct{}
}

func CreateBackendPromise() BackendPromise {
	return BackendPromise{
		backend: nil,
		updated: make(chan struct{}),
	}
}

func (bp *BackendPromise) Set(b backend.Backend) {
	if bp.backend != b {
		bp.backend = b
		go func() {
			bp.updated <- struct{}{}
		}()
	}
}

func (bp *BackendPromise) Await(ctx context.Context) (backend.Backend, error) {
	for {
		if bp.backend != nil {
			return bp.backend, nil
		}
		select {
		case <-bp.updated:
			continue
		case <-ctx.Done():
			return nil, status.Errorf(codes.Canceled, "backend retrieval canceled")
		}
	}
}
