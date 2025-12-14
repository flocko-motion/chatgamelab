# Build stage for Go server
FROM golang:1.25 AS build

WORKDIR /app

# Copy go module files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy git directory for version info
COPY .git .git
# Copy the rest of the server source
COPY . .

# Build the server binary with version info
RUN GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") && \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.GitCommit=${GIT_COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o cgl ./

# Runtime image (slim, no shell, based on Debian)
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy binary
COPY --from=build /app/cgl /app/cgl

# Directory for writable data
VOLUME ["/app/var"]

EXPOSE 3000

ENTRYPOINT ["/app/cgl"]
