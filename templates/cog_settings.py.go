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

package templates

const COG_SETTINGS_PY = `
import cogment as _cog
from types import SimpleNamespace
from typing import List

{{range $i, $proto := .Import.Proto -}}
import {{$proto}} as {{index $.Import.ProtoAlias $i}}
{{end -}}
{{- range .Import.Python -}}
import {{.}}
{{end}}

{{range .Import.Proto -}}
protolib = "{{.}}"
{{end}}


{{- range .ActorClasses}}
_{{.Id}}_class = _cog.ActorClass(
    id='{{.Id}}',
    config_type={{if .ConfigType}}{{.ConfigType}}{{else}}None{{end}},
    action_space={{.Action.Space}},
	{{- with .Observation}}
    observation_space={{.Space}},
    observation_delta={{if .Delta}}{{.Delta}}{{else}}{{.Space}}{{end}},
    observation_delta_apply_fn={{if .DeltaApplyFn}}{{.DeltaApplyFn.Python}}{{else}}_cog.delta_encoding._apply_delta_replace{{end}},
    {{end -}}    
    feedback_space=None,
    message_space=None
)
{{end}}

actor_classes = _cog.actor_class.ActorClassList(
{{- range .ActorClasses}}
    _{{.Id}}_class,
{{- end}}
)

env_class = _cog.EnvClass(
    id='env',
    config_type=None,
    message_space=None
)

trial = SimpleNamespace(
    config_type={{if .Trial}}{{if .Trial.ConfigType}}{{.Trial.ConfigType}}{{else}}None{{end}}{{else}}None{{end}},
)

# Environment
environment = SimpleNamespace(
    config_type={{if .Environment}}{{if .Environment.ConfigType}}{{.Environment.ConfigType}}{{else}}None{{end}}{{else}}None{{end}},
)


class ActionsTable:
{{- range .ActorClasses}}
    {{.Id}}: List[{{.Action.Space}}]
{{- end}}

    def __init__(self, trial):
{{- range $i, $ac := .ActorClasses}}
        self.{{$ac.Id}} = [{{$ac.Action.Space}}() for _ in range(trial.actor_counts[{{$i}}])]
{{- end}}

    def all_actions(self):
        return {{range $i, $ac := .ActorClasses}}{{if $i}} + {{end}}self.{{.Id}}{{end}}

`
