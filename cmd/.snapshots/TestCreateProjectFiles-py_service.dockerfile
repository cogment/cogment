FROM python:3.7-slim

WORKDIR /service

# Install dependencies
COPY requirements.txt ./
RUN pip install -r requirements.txt
RUN pip install cogment[generate]

# Copy the rest of the service sources
COPY . ./

# Generate code
RUN python -m cogment.generate

CMD ["python", "main.py"]

