# SPDX-License-Identifier: BSD-3-Clause

# Build Image
FROM docker.io/golang:1.26-trixie AS builder

ARG BRIDGE_VERSION=development
ARG BRIDGE_COMMIT=HEAD

ENV CGO_ENABLED=0

WORKDIR /go/src/app
COPY . .

RUN set -ex; \
    go build -ldflags="-s -w -X main.version=${BRIDGE_VERSION} -X main.commit=${BRIDGE_COMMIT}" -o /go/bin/alertmanager-icinga-bridge

# Final Image
FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /

COPY --from=builder /go/bin/alertmanager-icinga-bridge /

EXPOSE 8888

ENTRYPOINT ["/alertmanager-icinga-bridge"]
