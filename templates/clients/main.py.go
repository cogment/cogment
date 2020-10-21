// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

const MAIN_PY = `
{{ $config := . }}
import cog_settings

{{range .ActorClasses}}
{{- if $config.HasHumanByActorClass .Id}}from data_pb2 import {{.Id|pascalify}}Action{{end}}
{{- end}}

from cogment.client import Connection

# Callback received message
def handle_messages(sender, msg):
    print(f'Client received message - {msg} from sender {sender}')

# Create a connection to the Orchestrator serving this project
conn = Connection(cog_settings, "orchestrator:9000")

# Initiate a trial
trial = conn.start_trial(cog_settings.actor_classes.{{range .ActorClasses}}{{if $config.HasHumanByActorClass .Id }}{{.Id|snakeify}}{{- end}}
{{- end}})

# Perform actions, and get observations
{{- range .ActorClasses}}
{{- if $config.HasHumanByActorClass .Id }}
observation = trial.do_action({{.Id|pascalify}}Action(), on_message = handle_messages)
observation = trial.do_action({{.Id|pascalify}}Action(), on_message = handle_messages)
{{end -}}
{{end}}


# cleanup
trial.end()
`
