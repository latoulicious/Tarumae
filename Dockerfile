# Use an official Go runtime as a parent image
FROM golang:1.19-alpine AS builder

# Install youtube-dl
RUN apk add --no-cache youtube-dl

# Set the working directory in the container
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .

# Install any needed packages specified in go.mod
RUN go mod download

# Build the Go app
RUN go build -o main .

# Use a smaller base image for the final runtime
FROM alpine:latest

# Install youtube-dl
RUN apk add --no-cache youtube-dl

# Set the working directory in the container
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Copy the configuration file
COPY config.json .

# Run the application
CMD ["./main"]
