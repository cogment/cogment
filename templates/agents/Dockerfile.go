package agents

const DOCKERFILE = `
FROM python:3.7

ENV COGMENT_VERSION 0.2
ENV PYTHONPATH /app

RUN pip install cogment==$COGMENT_VERSION

WORKDIR /app

ADD agents/{{.|snakeify}} ./agents/{{.|snakeify}}
ADD cog_settings.py .
ADD *_pb2.py .

CMD ["python", "agents/{{.|snakeify}}/main.py"]
`
