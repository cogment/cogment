FROM python:3.7-slim

WORKDIR /service

# Install dependencies
COPY requirements.txt ./
COPY cogment.yaml ./
COPY *.proto ./

# Install dependencies
COPY requirements.txt ./
RUN pip install -r requirements.txt

COPY cogment.yaml *.proto ./
RUN python -m cogment.generate

# Copy the rest of the service sources
COPY . ./

CMD ["python", "main.py"]

