// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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

import { Context } from "@cogment/cogment-js-sdk";
import { useEffect, useState } from "react";

export const useActions = (cogSettings, actorName, actorClass) => {
  const [event, setEvent] = useState({
    observation: null,
    actions: null,
    messages: null,
    rewards: null,
    type: null,
    last: false
  });

  const [startTrial, setStartTrial] = useState(null);
  const [sendAction, setSendAction] = useState(null);

  //Set up the connection and register the actor only once, regardless of re-rendering
  useEffect(() => {
    const context = new Context(
      cogSettings,
      actorName,
    );

    context.registerActor(async (actorSession) => {
      actorSession.start();

      //Double arrow function here beause react will turn a single one into a lazy loaded function
      setSendAction(() => (action) => {
        actorSession.doAction(action);
      });

      for await (const event of actorSession.eventLoop()) {
        const eventUseActions = event;

        eventUseActions.last = event.type === 3;

        setEvent(eventUseActions);
      }
    }, actorName, actorClass)

    const endpoint = window.location.protocol + "//" + window.location.hostname + ":8080"
    const controller = context.getController(endpoint);

    //Need to output a function so that the user can start the trial when all actors are connected
    //Again, double arrow function cause react will turn a single one into a lazy loaded function
    setStartTrial(() => async () => {
      const trialId = await controller.startTrial();
      await context.joinTrial(trialId, endpoint, actorName);
    });
  }, [cogSettings, actorName, actorClass]);

  return [event, startTrial, sendAction];
};
