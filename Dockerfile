ARG goversion=1.24

FROM golang:${goversion}-alpine AS builder
RUN apk update --no-cache && \
    apk upgrade --no-cache && \
    apk add --no-cache build-base git curl
COPY go.mod /app/
WORKDIR /app
RUN go mod download
COPY . /app
RUN curl --fail --silent --show-error -L --output iamoidccredhelper https://gitlab.com/api/v4/projects/44551702/packages/generic/iam-oidc-credential-helper/v0.1.1/iamoidccredhelper_v0.1.1_linux_amd64 && \
    make build-api && \
    make build-job-executor && \
    make build-runner

FROM gcr.io/distroless/static-debian12:nonroot AS distroless-base
WORKDIR /app/

FROM distroless-base AS api
COPY --from=builder /app/apiserver .
USER nonroot
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 CMD [ "curl", "-f", "http://localhost:8000/health", "||", "exit", "1" ]
EXPOSE 8000
CMD ["./apiserver"]

FROM alpine:3.21 AS runner
RUN apk update --no-cache && \
    apk upgrade --no-cache && \
    apk add --no-cache git curl python3 py3-pip jq && \
    adduser tharsis -D && \
    addgroup docker && \
    adduser tharsis docker && \
    mkdir -p /app /opt/credhelpers && \
    chown tharsis:tharsis /app
WORKDIR /app/
COPY --from=builder /app/runner .
COPY --chmod=0755 --from=builder /app/iamoidccredhelper /opt/credhelpers/iamoidccredhelper
USER tharsis
HEALTHCHECK NONE
CMD ["./runner"]

FROM alpine:3.21 AS job-executor
WORKDIR /app/
COPY --from=builder /app/job .
RUN apk update --no-cache && \
    apk upgrade --no-cache && \
    apk add --no-cache git curl python3 py3-pip jq && \
    adduser tharsis -D && \
    chown tharsis:tharsis /app
USER tharsis
HEALTHCHECK NONE
CMD ["./job"]
