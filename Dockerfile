# SPDX-License-Identifier: BSD-3-Clause

# Build Image
FROM docker.io/golang:1.27-trixie AS builder

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN set -xe; \
    go mod download

ARG BRIDGE_VERSION=development

COPY . .
RUN set -ex; \
    make release "VERSION=${BRIDGE_VERSION}"

# Final Image
FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /

COPY --from=builder /go/src/app/dist/ /

EXPOSE 8888

ENTRYPOINT ["/alertmanager-icinga-bridge"]
