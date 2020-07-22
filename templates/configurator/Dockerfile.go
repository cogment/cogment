package configurator

const DOCKERFILE = `
FROM python:3.7

# Uncomment following and set specific version if required for only this service
#ENV COGMENT_VERSION 0.3.0a5
#RUN pip install cogment==$COGMENT_VERSION
# Comment following if above is used
ADD requirements.txt .
RUN pip install -r requirements.txt

WORKDIR /app

ADD configurator ./configurator/
ADD cog_settings.py .
ADD *_pb2.py .

CMD ["python", "configurator/main.py"]
`
