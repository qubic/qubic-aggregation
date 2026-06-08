FROM golang:1.26 AS builder
ENV CGO_ENABLED=0

# Build context is the monorepo root so the local `replace ../shared` directive
# in general-service/go.mod resolves correctly.
WORKDIR /src
COPY general-service /src/general-service
COPY shared /src/shared

WORKDIR /src/general-service
RUN go build -o "./bin/general-service" "./cmd/general-service"

FROM alpine
COPY --from=builder /src/general-service/bin/general-service /app/general-service

EXPOSE 8000
EXPOSE 8001
EXPOSE 9999

WORKDIR /app

ENTRYPOINT ["./general-service"]