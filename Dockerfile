# SPDX-License-Identifier: BSD-3-Clause

# Build Image
FROM docker.io/golang:latest as builder

ARG BRIDGE_VERSION=development
ARG BRIDGE_COMMIT=HEAD
ARG BRIDGE_DATE=latest

ENV CGO_ENABLED=0

WORKDIR /go/src/app
COPY . .

RUN set -ex; \
    go build -ldflags="-s -w -X main.version=${BRIDGE_VERSION} -X main.commit=${BRIDGE_COMMIT} -X main.date=${BRIDGE_DATE}" -o /go/bin/alertmanager-icinga-bridge

# Final Image
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /go/bin/alertmanager-icinga-bridge /

EXPOSE 8888

ENTRYPOINT ["/alertmanager-icinga-bridge"]
