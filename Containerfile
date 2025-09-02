# Stage 1: Build the Go binary using Red Hat's Go toolset image
FROM registry.access.redhat.com/ubi8/go-toolset:latest AS builder

# Set the working directory
WORKDIR /build

# Copy go.mod and go.sum to download dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application, statically linking it to ensure it runs on a minimal base image
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /tmp/helm-template-service .

# Stage 2: Create the final, minimal image
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

# Set the working directory
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /tmp/helm-template-service .

# Expose the port the application runs on
EXPOSE 8080

# Set the user to a non-root user for security
USER 1001

# Define the command to run the application
CMD ["./helm-template-service"]