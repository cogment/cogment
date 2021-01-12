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
        if "actions" in event:
            print("environment updating")
            observation = Observation()
            environment_session.produce_observations([("*", observation)])
        if "message" in event:
            (msg, sender) = event["message"]
            print(f"environment received message - {msg} from sender {sender}")

    print("environment end")

async def main():
    print("Environment service up and running.")

    context = cogment.Context(cog_settings=cog_settings, user_id="testit")

    context.register_environment(impl=environment)

    await context.serve_all_registered(port=9000)

if __name__ == '__main__':
    asyncio.run(main())
