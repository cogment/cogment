FROM cogment/orchestrator:v2.0.0

COPY cogment.yaml .
COPY *.proto .

CMD ["--params=cogment.yaml"]