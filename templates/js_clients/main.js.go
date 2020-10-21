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

package js_clients

const MAIN_JS = `
{{ $config := . }}
import {Connection} from 'cogment';
import cog_settings from './cog_settings';
import * as data_pb2 from './data_pb.js';

const human_class = cog_settings.actor_classes.{{range .ActorClasses}}{{if $config.HasHumanByActorClass .Id }}{{.Id|snakeify}}{{- end}}
{{- end}};

const hostname = window.location.hostname;

let trial, observation, human_id;
let endpoint = ` + "`" + `http://${hostname}:8088` + "`" + `;

async function start() {

    const conn = new Connection(cog_settings, endpoint);

    if (!trial) {
        try {
            console.log("start trial");
            trial = await conn.start_trial(cog_settings.actor_classes.{{range .ActorClasses}}{{if $config.HasHumanByActorClass .Id }}{{.Id|snakeify}}{{- end}}
{{- end}});
            human_id = trial.id.charAt(trial.id.length - 1);
        } catch (e) {
            console.error(e);
            return
        }
    }

    console.log(` + "`" + `Client Trial now established, id: ${trial.id} + human id: ${human_id}` + "`" + `);

	const action = new human_class.action_space();

	observation = await trial.do_action(action).catch(e => {
		console.log(e)
	});
	observation = await trial.do_action(action).catch(e => {
		console.log(e)
	});
	console.log('observation: ' + observation)

	trial.end()

}

window.addEventListener("DOMContentLoaded", (event) => {
    console.log("Dom loaded");
    start();
});
`