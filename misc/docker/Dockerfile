FROM golang:1.25 AS build

RUN apt-get update \
  && apt-get install -y make git

RUN --mount=type=cache,target=/go/pkg/mod/ \
  --mount=type=bind,source=go.sum,target=go.sum \
  --mount=type=bind,source=go.mod,target=go.mod \
  go mod download -x

WORKDIR /src

COPY . /src

ARG GOCACHE=/root/.cache/go-build

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=cache,target=./tools \
    make goreleaser GORELEASER_ARGS="build --clean --auto-snapshot --single-target"

FROM alpine:3.21 AS runtime

RUN apk add \
    ca-certificates \
    openssl \
    pandoc \
    gcompat \
  && update-ca-certificates

COPY --from=build /src/dist/xolo-server_linux_amd64_v1/xolo-server /usr/local/bin/xolo-server
COPY --from=build /src/dist/xolo-client_linux_amd64_v1/xolo-client /usr/local/bin/xolo-client

CMD ["/usr/local/bin/xolo-server"]