# Build stage for Go server
FROM golang:1.25 AS build

WORKDIR /app

# Copy go module files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the server source
COPY . .

# Build the server binary
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o webapp-server ./

# Runtime image (slim, no shell, based on Debian)
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy binary
COPY --from=build /app/webapp-server /app/webapp-server

# Directory for writable data
VOLUME ["/app/var"]

EXPOSE 3000

ENTRYPOINT ["/app/webapp-server"]
