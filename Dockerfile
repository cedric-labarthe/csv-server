# syntax=docker/dockerfile:1.7
# Development image for csv-server with full Go toolchain available.
FROM golang:1.26.0-alpine3.22

# Create a non-root user.
RUN addgroup -S app && adduser -S app -G app

# Configure writable Go caches for the non-root user.
ENV APP_HOME=/workspace \
    GOPATH=/home/app/go \
    GOCACHE=/home/app/.cache/go-build \
    GOMODCACHE=/home/app/go/pkg/mod \
    PATH=/usr/local/go/bin:/home/app/go/bin:${PATH}

WORKDIR ${APP_HOME}

# Copy module files first to maximize dependency layer caching.
COPY --chown=app:app go.mod go.sum ./
RUN go mod download

# Keep full source available in-image; docker-compose bind mount overrides this in dev.
COPY --chown=app:app . .

RUN mkdir -p /home/app/.cache/go-build /home/app/go/pkg/mod \
    && chown -R app:app /home/app ${APP_HOME}

USER app

# Default server port.
EXPOSE 8080

# Default command for development; override in docker-compose as needed.
CMD ["go", "run", "./cmd/server"]
