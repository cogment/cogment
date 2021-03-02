import cog_settings
from data_pb2 import Observation

import cogment

import asyncio

async def environment(environment_session):
    print("environment starting")
    # Create the initial observaton
    observation = Observation()

    # Start the trial and send that observation to all actors
    environment_session.start([("*", observation)])

    async for event in environment_session.event_loop():
        if event.actions:
            actions = event.actions
            print(f"environment received the actions")
            for actor, action in zip(environment_session.get_active_actors(), actions):
                print(f" actor '{actor.actor_name}' did action '{action}'")
            observation = Observation()
            if event.type == cogment.EventType.ACTIVE:
                # The trial is active
                environment_session.produce_observations([("*", observation)])
            else:
                # The trial termination has been requested
                environment_session.end([("*", observation)])
        for message in event.messages:
            print(f"environment received a message from '{message.sender_name}': - '{message.payload}'")

    print("environment end")

async def main():
    print("Environment service starting...")

    context = cogment.Context(cog_settings=cog_settings, user_id="testit")

    context.register_environment(impl=environment)

    await context.serve_all_registered(port=9000)

if __name__ == '__main__':
    asyncio.run(main())

