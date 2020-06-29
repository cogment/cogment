package envs

const DOCKERFILE = `
FROM python:3.7

ENV COGMENT_VERSION 0.2
ENV PYTHONPATH /app

RUN pip install cogment==$COGMENT_VERSION

WORKDIR /app

ADD envs ./envs
ADD cog_settings.py .
ADD *_pb2.py .

CMD ["python", "envs/main.py"]
`
