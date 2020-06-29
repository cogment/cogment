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
    feedback_space=None
)
{{end}}

actor_classes = _cog.actor_class.ActorClassList(
{{- range .ActorClasses}}
    _{{.Id}}_class,
{{- end}}
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

{{ range .ActorClasses}}
class {{.Id}}_ObservationProxy(_cog.env_service.ObservationProxy):
{{- with .Observation}}
    @property
    def snapshot(self) -> {{.Space}}:
        return self._get_snapshot({{.Space}})

    @snapshot.setter
    def snapshot(self, v):
        self._set_snapshot(v)

    @property
    def delta(self) -> {{if .Delta}}{{.Delta}}{{else}}{{.Space}}{{end}}:
        return self._get_delta({{if .Delta}}{{.Delta}}{{else}}{{.Space}}{{end}})

    @delta.setter
    def delta(self, v):
        self._set_delta(v)
{{end -}}
{{end}}

class ObservationsTable:
{{- range .ActorClasses}}
    {{.Id}}: List[{{.Id}}_ObservationProxy]
{{- end}}

    def __init__(self, trial):
{{- range $i, $ac := .ActorClasses}}
        self.{{$ac.Id}} = [{{$ac.Id}}_ObservationProxy() for _ in range(trial.actor_counts[{{$i}}])]
{{- end}}

    def all_observations(self):
        return {{range $i, $ac := .ActorClasses}}{{if $i}} + {{end}}self.{{.Id}}{{end}}
`
