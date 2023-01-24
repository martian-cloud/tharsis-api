ARG goversion

FROM golang:${goversion}-alpine as builder
RUN apk add --no-cache build-base git
COPY go.mod /app/
WORKDIR /app
RUN go mod download
COPY . /app
RUN make build-api
RUN make build-job-executor

FROM gcr.io/distroless/static-debian11:nonroot as tharsis-base
WORKDIR /app/

FROM tharsis-base AS api
COPY --from=builder /app/apiserver .
USER nonroot
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 CMD [ "curl", "-f", "http://localhost:8000/health", "||", "exit", "1" ]
EXPOSE 8000
CMD ["./apiserver"]

FROM tharsis-base AS job-executor
COPY --from=builder /app/job .
USER nonroot
HEALTHCHECK NONE
CMD ["./job"]
