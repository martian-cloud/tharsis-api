# syntax=docker/dockerfile:1

# Build stages fan out from builder-base so that targeting one image compiles only that image's
# binary.
FROM golang:1.26.7-alpine@sha256:bf9573d7c1d2b09992e4f893ea1ef30842854846bdb8ae390468f95ea6b09062 AS builder-base

# The frontend toolchain lives in build-tharsis, the only stage that needs it.
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    build-base \
    git && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Download the credential helper in its own stage, independent of the source tree, so editing code
# does not re-fetch it and the images that do not ship it never fetch it at all.
FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS credhelper
RUN apk add --no-cache curl && \
    curl --fail --silent --show-error -L \
    --output /iamoidccredhelper \
    https://gitlab.com/api/v4/projects/44551702/packages/generic/iam-oidc-credential-helper/v0.2.0/iamoidccredhelper_v0.2.0_linux_amd64 && \
    chmod +x /iamoidccredhelper

# The cache mounts survive --no-cache, which invalidates layers but not mounted caches, so a forced
# rebuild still reuses the module and compiler caches.
FROM builder-base AS build-runner
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    make build-runner

FROM builder-base AS build-job-executor
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    make build-job-executor

FROM builder-base AS build-tharsis
RUN apk add --no-cache nodejs npm && rm -rf /var/cache/apk/*
# Increase default memory limit
ENV NODE_OPTIONS="--max-old-space-size=4096"
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    make build-tharsis

FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1 AS distroless-base
WORKDIR /app/

FROM distroless-base AS tharsis
COPY --from=build-tharsis --chown=nonroot:nonroot /app/apiserver .
USER 65532:65532
EXPOSE 8000
CMD ["./apiserver"]

FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS runner
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    git \
    curl \
    python3 \
    py3-pip \
    jq && \
    adduser tharsis -D -u 1001 -g 1001 && \
    mkdir -p /app /opt/credhelpers && \
    chown -R tharsis:tharsis /app /opt/credhelpers && \
    addgroup docker && \
    adduser tharsis docker && \
    find /usr /bin /sbin -perm /6000 -type f -exec rm -f {} \; && \
    rm -rf /var/cache/apk/* && \
    rm -rf /etc/apk && \
    rm -rf /usr/share/man /usr/share/doc /tmp/* /var/tmp/*

WORKDIR /app/
COPY --from=build-runner --chown=tharsis:tharsis /app/runner .
COPY --from=credhelper --chown=tharsis:tharsis --chmod=0755 /iamoidccredhelper /opt/credhelpers/iamoidccredhelper
USER tharsis
HEALTHCHECK NONE
CMD ["./runner"]

FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS job-executor
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    git \
    curl \
    python3 \
    py3-pip \
    jq && \
    adduser tharsis -D -u 1001 -g 1001 && \
    mkdir -p /app /etc && \
    chown -R tharsis:tharsis /app && \
    # Maintain backward compatibility for users that are installing packages such as the aws cli w/o a virtual env using pip3
    echo "[global]" > /etc/pip.conf && \
    echo "break-system-packages = true" >> /etc/pip.conf && \
    find /usr /bin /sbin -perm /6000 -type f -exec rm -f {} \; && \
    rm -rf /var/cache/apk/* && \
    rm -rf /usr/share/man /usr/share/doc /tmp/* /var/tmp/*

WORKDIR /app/
COPY --from=build-job-executor --chown=tharsis:tharsis /app/job .
USER tharsis
HEALTHCHECK NONE
CMD ["./job"]
