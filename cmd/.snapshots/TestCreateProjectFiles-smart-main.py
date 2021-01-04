import cog_settings
from data_pb2 import SmartAction

import cogment

import asyncio

async def smart_impl(actor_session):
    actor_class = cog_settings.actor_classes.smart

    actor_session.start()

    async for event in actor_session.event_loop():
        if "observation" in event:
            observation = event["observation"]
            print("Smart decide")
            action = SmartAction()
            actor_session.do_action(action)
        if "reward" in event:
            reward = event["reward"]
            print("Smart reward")
        if "message" in event:
            (msg, sender) = event["message"]
            print(f"Smart received message - {msg} from sender {sender}")

async def main():
    print("Smart service up and running.")

    context = cogment.Context(cog_settings=cog_settings, user_id="testit")

    context.register_actor(
        impl=smart_impl,
        impl_name="smart_impl",
        actor_classes=["smart"])

    await context.serve_all_registered(port=9000)

if __name__ == '__main__':
    asyncio.run(main())

