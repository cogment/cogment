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

package agents

const MAIN_PY = `
import cog_settings
from data_pb2 import {{.|pascalify}}Action

from cogment import Agent, GrpcServer

class {{.|pascalify}}(Agent):
    VERSIONS = {"{{.|snakeify}}": "1.0.0"}
    actor_class = cog_settings.actor_classes.{{.|snakeify}}

    def decide(self, observation: cog_settings.actor_classes.{{.|snakeify}}.observation_space):
        print("{{.|pascalify}} decide")
        action = {{.|pascalify}}Action()
        return action

    def reward(self, reward):
        print("{{.|pascalify}} reward")

    def on_message(self, sender, msg):
        if msg:
            print(f'Agent {self.id_in_class} received message - {msg} from sender {sender}')

    def end(self):
        print("{{.|pascalify}} end")


if __name__ == '__main__':
    server = GrpcServer({{.|pascalify}}, cog_settings)
    server.serve()
`
