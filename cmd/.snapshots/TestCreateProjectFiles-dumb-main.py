import cog_settings
from data_pb2 import DumbAction

import cogment

import asyncio

async def dumb_impl(actor_session):
    actor_class = cog_settings.actor_classes.dumb

    actor_session.start()

    async for event in actor_session.event_loop():
        if "observation" in event:
            observation = event["observation"]
            print("Dumb decide")
            action = DumbAction()
            actor_session.do_action(action)
        if "reward" in event:
            reward = event["reward"]
            print("Dumb reward")
        if "message" in event:
            (msg, sender) = event["message"]
            print(f"Dumb received message - {msg} from sender {sender}")

async def main():
    print("Dumb service up and running.")

    context = cogment.Context(cog_settings=cog_settings, user_id="testit")

    context.register_actor(
        impl=dumb_impl,
        impl_name="dumb_impl",
        actor_classes=["dumb"])

    await context.serve_all_registered(port=9000)

if __name__ == '__main__':
    asyncio.run(main())

