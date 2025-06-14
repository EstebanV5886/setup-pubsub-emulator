# Use Alpine + Go base image
FROM golang:1.24-alpine

# Set working directory
WORKDIR /app

# Install git (required for go get, esp. with private repos)
RUN apk add --no-cache git

# Copy go.mod and go.sum separately to leverage Docker cache
COPY go.mod go.sum ./

# Download dependencies early (cache optimization)
RUN go mod download

# Copy the rest of the source code
COPY . .

# Tidy and build
RUN go mod tidy
RUN go build -o setup main.go

# Entrypoint
CMD ["./setup"]