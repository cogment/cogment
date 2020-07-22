package configurator

const  CONFIGURATOR_PY = `{{ $config := . }}

import traceback
from cogment import TrialHooks, GrpcServer
from types import SimpleNamespace as ns

// endpoints
{{- range .ActorClasses}}{{- if $config.HasAiByActorClass .Id }}{{.Id|tocaps}}_URL = 'grpc://{{.Id}}:9000'{{end}}
{{end}}

class Supervisor(TrialHooks):
    VERSIONS = {"poker": "1.0.0"}

    @staticmethod
    def pre_trial(trial_id, user_id, trial_params):

        actor_settings = {
{{- range .ActorClasses}}{{- if $config.HasAiByActorClass .Id }}
            "{{.Id}}": ns(
            actor_class='{{.Id}}',
            end_point={{.Id|tocaps}}_URL,
            config=None
            ),
{{end}}
{{end}}
        }


        try:
            trial_config = trial_params.trial_config

{{- range .ActorClasses}}
{{- if $config.HasHumanByActorClass .Id }}
            actors = [ns(
                actor_class='{{.Id}}',
                endpoint="human",
                config=None
            )]
{{end -}}
{{end}}

            # modify following to retrieve config info and create actors list 
            #for ??? in trial_config.env_config.???:
            #    actors.append(actor_settings[???])

            trial_params.actors = actors

            trial_params.environment.config = trial_config.env_config

            return trial_params
        except Exception:
            traceback.print_exc()
            raise

if __name__ == '__main__':
    server = GrpcServer(Configurator, cog_settings)
    server.serve()
`
