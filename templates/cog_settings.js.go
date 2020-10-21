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

const COG_SETTINGS_JS = `
import {apply_delta_replace} from 'cogment/delta_encoding'
{{range $i, $proto := .Import.Proto -}}
import * as {{index $.Import.ProtoAlias $i}} from './{{$proto}}.js';
{{end -}}
{{- range .Import.Javascript -}}
import * as {{.}} from './{{.}}.js';
{{end}}

{{- range .ActorClasses}}
const _{{.Id}}_class = {
  id: '{{.Id}}',
  config_type: {{if .ConfigType}}{{.ConfigType}}{{else}}null{{end}},
  action_space: {{.Action.Space}},
  observation_space: {{.Observation.Space}},
  observation_delta: {{if .Observation.Delta}}{{.Observation.Delta}}{{else}}{{.Observation.Space}}{{end}},
  observation_delta_apply_fn: {{if .Observation.DeltaApplyFn}}{{.Observation.DeltaApplyFn.Javascript}}{{else}}apply_delta_replace{{end}},
  feedback_space: null,
  message_space: null
};
{{end}}


const settings = {
  actor_classes: {
  {{- range .ActorClasses}}
    {{.Id}}: _{{.Id}}_class,
  {{- end}}
  },

  trial: {
    config_type: {{if .Trial.ConfigType}}{{.Trial.ConfigType}}{{else}}null{{end}},
  },

  environment: {
    config_type: {{if .Environment.ConfigType}}{{.Environment.ConfigType}}{{else}}null{{end}},
  },

  env_class: {
    id: 'env',
    config_type: null,
    message_space: null
  }

};

export default settings;
`
