// SPDX-License-Identifier: BSD-3-Clause

FROM docker.io/golang:latest as builder

WORKDIR /go/src/app
COPY . .

RUN CGO_ENABLED=0 go build -o /go/bin/alertmanager-icinga-bridge

FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /go/bin/alertmanager-icinga-bridge /

EXPOSE 8888

ENTRYPOINT ["/alertmanager-icinga-bridge"]
