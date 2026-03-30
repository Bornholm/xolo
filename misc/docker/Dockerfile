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

FROM alpine:3.23 AS runtime

RUN apk add \
    ca-certificates \
    openssl \
    gcompat \
  && update-ca-certificates

COPY --from=build /src/dist/xolo-server_linux_amd64_v1/xolo-server /usr/local/bin/xolo-server
COPY --from=build /src/dist/xolo-plugin-time-restriction_linux_amd64_v1/xolo-plugin-time-restriction /plugins/time-restriction
COPY --from=build /src/dist/xolo-plugin-smart-model_linux_amd64_v1/xolo-plugin-smart-model /plugins/smart-model
COPY --from=build /src/dist/xolo-plugin-dummy-model_linux_amd64_v1/xolo-plugin-dummy-model /plugins/dummy-model

ENV XOLO_STORAGE_DATABASE_DSN=/data/data.sqlite
ENV XOLO_PLUGINS_DIR=/plugins

CMD ["/usr/local/bin/xolo-server"]