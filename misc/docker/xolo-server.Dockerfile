FROM alpine:3.23 AS runtime

RUN apk add \
    ca-certificates \
    openssl \
    gcompat \
  && update-ca-certificates

COPY xolo-server /usr/local/bin/xolo-server

RUN mkdir -p /plugins
COPY xolo-plugin-time-restriction /plugins/time-restriction
COPY xolo-plugin-smart-model /plugins/xolo-plugin-smart-model
COPY xolo-plugin-dummy-model /plugins/xolo-plugin-dummy-model

ENV XOLO_STORAGE_DATABASE_DSN=/data/data.sqlite
ENV XOLO_PLUGINS_DIR=/plugins

VOLUME /data

EXPOSE 3002

CMD ["/usr/local/bin/xolo-server"]
