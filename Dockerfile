ARG goversion

FROM golang:${goversion}-alpine as builder
ENV GOPRIVATE gitlab.com/infor-cloud/martian-cloud/tharsis/*
ENV GO111MODULE on
RUN apk add build-base git
ADD go.mod /app/go.mod
RUN cd /app && go mod download
ADD . /app
WORKDIR /app
RUN make build-api
RUN make build-job-executor

FROM alpine:latest as tharsis-base
WORKDIR /app/

FROM tharsis-base AS api
COPY --from=builder /app/apiserver .
EXPOSE 8000
CMD ["./apiserver"]

FROM tharsis-base AS job-executor
COPY --from=builder /app/job .
CMD ["./job"]
