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
