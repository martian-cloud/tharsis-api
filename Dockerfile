FROM golang:1.24-alpine@sha256:fc2cff6625f3c1c92e6c85938ac5bd09034ad0d4bc2dfb08278020b68540dbb5 AS builder

RUN apk update --no-cache && \
    apk add --no-cache \
    build-base \
    git \
    curl \
    nodejs \
    npm && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Download credential helper
RUN curl --fail --silent --show-error -L \
    --output iamoidccredhelper \
    https://gitlab.com/api/v4/projects/44551702/packages/generic/iam-oidc-credential-helper/v0.1.1/iamoidccredhelper_v0.1.1_linux_amd64 && \
    chmod +x iamoidccredhelper

# Build binaries
RUN make build-tharsis && \
    make build-job-executor && \
    make build-runner

FROM gcr.io/distroless/static-debian12:nonroot@sha256:e8a4044e0b4ae4257efa45fc026c0bc30ad320d43bd4c1a7d5271bd241e386d0 AS distroless-base
WORKDIR /app/

FROM distroless-base AS tharsis
COPY --from=builder --chown=nonroot:nonroot /app/apiserver .
USER 65532:65532
EXPOSE 8000
CMD ["./apiserver"]

FROM alpine:3.21@sha256:b6a6be0ff92ab6db8acd94f5d1b7a6c2f0f5d10ce3c24af348d333ac6da80685 AS runner
RUN apk update --no-cache && \
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
COPY --from=builder --chown=tharsis:tharsis /app/runner .
COPY --from=builder --chown=tharsis:tharsis --chmod=0755 /app/iamoidccredhelper /opt/credhelpers/iamoidccredhelper
USER tharsis
HEALTHCHECK NONE
CMD ["./runner"]

FROM alpine:3.21@sha256:b6a6be0ff92ab6db8acd94f5d1b7a6c2f0f5d10ce3c24af348d333ac6da80685 AS job-executor
RUN apk update --no-cache && \
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
    rm -rf /etc/apk && \
    rm -rf /usr/share/man /usr/share/doc /tmp/* /var/tmp/*

WORKDIR /app/
COPY --from=builder --chown=tharsis:tharsis /app/job .
USER tharsis
HEALTHCHECK NONE
CMD ["./job"]
