# Stage 1: Build the frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build the backend
FROM golang:1.24-alpine AS backend-builder
RUN apk add --no-cache build-base sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy static files from frontend build
COPY --from=frontend-builder /app/frontend/public ./frontend/public
# Build the binary with SQLite support (requires CGO)
# Use musl tag to work around Alpine/musl libc limitations with pread64/pwrite64
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -tags musl -o pulse .

# Stage 3: Final minimal image
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /root/
# Copy binary from builder
COPY --from=backend-builder /app/pulse .
# Copy static files from builder (for server to serve)
COPY --from=backend-builder /app/frontend/public ./frontend/public
# Create data directory for SQLite
RUN mkdir -p /root/data

EXPOSE 8080
ENV PORT=8080
ENV GIN_MODE=release

CMD ["./pulse"]
