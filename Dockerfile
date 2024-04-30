ARG goversion

FROM golang:${goversion}-alpine as builder
RUN apk add --no-cache build-base git curl
COPY go.mod /app/
WORKDIR /app
RUN go mod download
COPY . /app
RUN curl https://gitlab.com/api/v4/projects/44551702/packages/generic/iam-oidc-credential-helper/v0.0.1/iamoidccredhelper_v0.0.1_linux_amd64 -L --output iamoidccredhelper
RUN make build-api
RUN make build-job-executor
RUN make build-runner

FROM gcr.io/distroless/static-debian11:nonroot as distroless-base
WORKDIR /app/

FROM distroless-base AS api
COPY --from=builder /app/apiserver .
USER nonroot
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 CMD [ "curl", "-f", "http://localhost:8000/health", "||", "exit", "1" ]
EXPOSE 8000
CMD ["./apiserver"]

FROM distroless-base AS runner
COPY --from=builder /app/runner .
COPY --chmod=0755 --from=builder /app/iamoidccredhelper /opt/credhelpers/iamoidccredhelper
USER nonroot
HEALTHCHECK NONE
CMD ["./runner"]

FROM alpine:3.17 AS job-executor
WORKDIR /app/
COPY --from=builder /app/job .
RUN apk add --no-cache git curl python3 py3-pip jq && \
    apk --no-cache upgrade && \
    adduser tharsis -D && \
    chown tharsis:tharsis /app
USER tharsis
HEALTHCHECK NONE
CMD ["./job"]
