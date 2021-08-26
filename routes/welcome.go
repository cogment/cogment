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

package routes

import (
	"net/http"
)

type message struct {
	Message string `json:"message,omitempty"`
}

var welcomeMessage = message{"Welcome to the model registry API"}

func welcome() http.HandlerFunc {
	return errorHandler(func(w http.ResponseWriter, r *http.Request) error {
		err := encodeJSONResponse(w, welcomeMessage)
		if err != nil {
			return err
		}
		w.WriteHeader(http.StatusOK)
		return nil
	})
}
