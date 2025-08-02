# Stage 1: Build the Go backend
FROM golang:1.19-alpine AS builder

WORKDIR /app

# Copy the Go module files and download dependencies
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy the backend source code
COPY backend/ ./

# Build the backend application
RUN CGO_ENABLED=0 GOOS=linux go build -o /dashboard-backend

# Stage 2: Create the final image
FROM scratch

WORKDIR /app

# Copy the compiled backend from the builder stage
COPY --from=builder /dashboard-backend .

# Copy the frontend static files
COPY frontend/ /app/frontend/

# Expose the port the server listens on
EXPOSE 8080

# Set the entrypoint for the container
ENTRYPOINT ["/app/dashboard-backend"]
