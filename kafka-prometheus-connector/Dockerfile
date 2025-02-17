FROM golang:1.24

WORKDIR /app

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    librdkafka-dev \
    && rm -rf /var/lib/apt/lists/*

COPY . ./

RUN go mod download
RUN go build -o kafka-prometheus-connector .

EXPOSE 8080
CMD ["./kafka-prometheus-connector"]