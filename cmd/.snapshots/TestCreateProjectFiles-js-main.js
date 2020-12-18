
import {Connection} from 'cogment';
import cog_settings from './cog_settings';
import * as data_pb2 from './data_pb.js';

const human_class = cog_settings.actor_classes.master;

const hostname = window.location.hostname;

let trial, observation, human_id;
let endpoint = ` + "`" + `http://${hostname}:8088` + "`" + `;

async function start() {

    const conn = new Connection(cog_settings, endpoint);

    if (!trial) {
        try {
            console.log("start trial");
            trial = await conn.start_trial(cog_settings.actor_classes.master);
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

