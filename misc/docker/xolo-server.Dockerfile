FROM alpine:3.23 AS runtime

RUN apk add \
    ca-certificates \
    openssl \
    gcompat \
  && update-ca-certificates

COPY xolo-server /usr/local/bin/xolo-server

RUN mkdir -p /plugins
COPY xolo-plugin-time-restriction /plugins/time-restriction
COPY xolo-plugin-dummy-model /plugins/dummy-model
COPY xolo-plugin-fuzzy-evaluator /plugins/fuzzy-evaluator
COPY xolo-plugin-request-evaluator /plugins/request-evaluator
COPY xolo-plugin-script-processor /plugins/script-processor
COPY xolo-plugin-pseudonymizer /plugins/pseudonymizer
COPY xolo-plugin-mcp-bridge /plugins/mcp-bridge

ENV XOLO_STORAGE_DATABASE_DSN=/data/data.sqlite
ENV XOLO_PLUGINS_DIR=/plugins

VOLUME /data
VOLUME /root/.cache/go-anon

EXPOSE 3002

CMD ["/usr/local/bin/xolo-server"]
