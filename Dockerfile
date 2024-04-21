# Build stage for React frontend
FROM node:20 as build-node
WORKDIR /app
COPY client/package.json client/package-lock.json ./
RUN npm install
COPY client/ ./
RUN echo "$(date +%y%m%d).$(printf '%04d' $(date +%s) | tail -c 4)" > ./src/version.js

RUN npm run build

# Build stage for Go server
FROM golang:latest as build-go
WORKDIR /go/src/webapp-server
COPY server/ ./
RUN go get -d -v
# RUN go build -o webapp-server
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o webapp-server

# RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webapp-server .


# Final stage: Copy the React build and Go binary into a new image
FROM debian:latest
WORKDIR /app
# Install ca-certificates
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
# Install binaries
COPY --from=build-node /app/build /app/html
COPY --from=build-go /go/src/webapp-server/webapp-server /app/
RUN mkdir -p /app/var
VOLUME ["/app/var"]
EXPOSE 3000
CMD ["/app/webapp-server"]
