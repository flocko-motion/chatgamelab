# Build stage for Go server
FROM golang:1.25 AS build

WORKDIR /app

# Copy go module files first for better caching
COPY server/go.mod server/go.sum ./
RUN go mod download

# Copy the rest of the server source
COPY server/ ./

# Build the server binary with version info
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.GitCommit=${GIT_COMMIT} -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o cgl ./

# Runtime image (slim, no shell, based on Debian)
FROM gcr.io/distroless/base-debian13

WORKDIR /app

# Copy binary
COPY --from=build /app/cgl /app/cgl

# Directory for writable data
VOLUME ["/app/var"]

# Expose port (will be overridden by PORT_BACKEND env var)
EXPOSE 3000

ENTRYPOINT ["/app/cgl"]
