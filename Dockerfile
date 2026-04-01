# ===== Stage 1: Build Frontend =====
FROM node:20-alpine AS frontend-builder

WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --production=false
COPY frontend/ ./
RUN npm run build
# Output: /build/backend/static/

# ===== Stage 2: Build Go Binary =====
FROM golang:1.25-alpine AS backend-builder

WORKDIR /build/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# Copy built frontend static files into backend/static for embed
COPY --from=frontend-builder /build/backend/static/ ./static/

# Build static binary (pure Go, no CGO needed for modernc.org/sqlite)
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /sage-server .

# ===== Stage 3: Production Image =====
FROM alpine:3.21

# Install ca-certificates for external API calls (exchange rates)
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S sage && adduser -S sage -G sage

# Create data directory
RUN mkdir -p /data && chown sage:sage /data

# Copy binary
COPY --from=backend-builder /sage-server /usr/local/bin/sage-server

# Switch to non-root user
USER sage

# Expose port
EXPOSE 8321

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:8321/api/auth/status || exit 1

# Environment defaults
ENV SAGE_DB_PATH=/data/sage.db \
    SAGE_PORT=8321

# Data volume
VOLUME ["/data"]

ENTRYPOINT ["sage-server"]
