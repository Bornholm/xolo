FROM alpine:3.23 AS runtime

RUN apk add \
    ca-certificates \
    openssl \
    gcompat \
  && update-ca-certificates

COPY xolo-server /usr/local/bin/xolo-server

ENV XOLO_STORAGE_DATABASE_DSN=/data/data.sqlite

VOLUME /data

EXPOSE 3002

CMD ["/usr/local/bin/xolo-server"]
