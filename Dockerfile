FROM golang:1.21 AS build-stage
WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY internal ./internal
COPY cmd/jaeger-postgresql ./cmd

# # Build
RUN CGO_ENABLED=0 GOOS=linux go build -C cmd -o ../jaeger-postgres

FROM alpine:3.19

COPY --from=build-stage --chmod=100k /app/jaeger-postgres ./

EXPOSE 12345 12346

CMD ["./jaeger-postgres"]
