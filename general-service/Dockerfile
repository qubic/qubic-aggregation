FROM golang:1.26 AS builder
ENV CGO_ENABLED=0

WORKDIR /src
COPY . /src

RUN go build -o "./bin/general-service" "./cmd/general-service"

FROM alpine
COPY --from=builder /src/bin/general-service /app/general-service

EXPOSE 8000
EXPOSE 8001
EXPOSE 9999

WORKDIR /app

ENTRYPOINT ["./general-service"]