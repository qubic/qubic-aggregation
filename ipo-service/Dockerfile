FROM golang:1.26 AS builder
ENV CGO_ENABLED=0

WORKDIR /src
COPY . /src

RUN go build -o "./bin/ipo-service" "./cmd/ipo-service"

FROM alpine
COPY --from=builder /src/bin/ipo-service /app/ipo-service

EXPOSE 8000
EXPOSE 8001
EXPOSE 9999

WORKDIR /app

ENTRYPOINT ["./ipo-service"]