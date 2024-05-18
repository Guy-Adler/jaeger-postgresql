# Jaeger-PostgresQL

Save trace data to a PostgreSQL database.

## Local Development:

### Requirements:
- go version > 1.21
- docker version > 26.0.0

### Developing & running the project locally
1. Install the dependencies using `go mod install`
2. To run the project locally, including a temporary DB and an all-in-one jaeger configuration, run `docker compose watch`.
3. To build the project, run `docker build .`
