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

package constants

const (
	DirectoryEndpointKey  = "directory_endpoint"
	DirectoryEndpointEnv  = "COGMENT_DIRECTORY_ENDPOINT"
	DirectoryEndpointDesc = "Directory service gRPC endpoint"

	DirectoryAuthTokenKey  = "directory_authentication_token"
	DirectoryAuthTokenEnv  = "COGMENT_DIRECTORY_AUTHENTICATION_TOKEN"
	DirectoryAuthTokenDesc = "Authentication token for directory services"

	DirectoryRegistrationHostKey       = "directory_registration_host"
	DirectoryRegistrationPropertiesKey = "directory_registration_properties"

	LogLevelKey  = "log_level"
	LogLevelEnv  = "COGMENT_LOG_LEVEL"
	LogLevelDesc = "Minimum logging level (trace, debug, info, warning, error, off)"

	LogFileKey  = "log_file"
	LogFileEnv  = "COGMENT_LOG_FILE"
	LogFileDesc = "Log file name"

	VerboseKey      = "verbose"
	VerboseShortKey = "v"
	VerboseDesc     = "Increase verbosity of output"

	QuietKey      = "quiet"
	QuietShortKey = "q"
	QuietDesc     = "Decrease verbosity of output"
)
