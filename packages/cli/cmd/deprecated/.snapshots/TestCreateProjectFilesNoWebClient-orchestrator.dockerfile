FROM cogment/orchestrator:v1.0

COPY cogment.yaml .
COPY *.proto .

CMD ["--params=cogment.yaml"]
