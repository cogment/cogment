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

package controller

// This file omplements the openapi.Typer interface on the `controller` types to make the OpenAPI spec looks better

func (*TrialParams) TypeName() string       { return "TrialParams" }
func (*DatalogParams) TypeName() string     { return "DatalogParams" }
func (*EnvironmentParams) TypeName() string { return "EnvironmentParams" }
func (*ActorParams) TypeName() string       { return "ActorParams" }
func (*TrialInfo) TypeName() string         { return "TrialInfo" }
