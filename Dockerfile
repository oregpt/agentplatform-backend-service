FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o backend-service ./cmd/main.go

# Use a smaller image for the final container
FROM alpine:3.17

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/backend-service .

# Expose the port the service runs on
# Use the PORT environment variable that Cloud Run provides
EXPOSE 8080

# Command to run the executable
CMD ["./backend-service"]
