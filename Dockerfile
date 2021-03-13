FROM golang:1-alpine AS build

COPY . /build
RUN apk add ca-certificates make && \
    cd /build && \
      make

RUN mkdir -p /rootfs/bin && \
      cp /build/bin/graphql_exporter /rootfs/bin/ && \
    mkdir -p /rootfs/etc && \
      echo "nogroup:*:10000:nobody" > /rootfs/etc/group && \
      echo "nobody:*:10000:10000:::" > /rootfs/etc/passwd && \
    mkdir -p /rootfs/etc/ssl/certs && \
      cp /etc/ssl/certs/ca-certificates.crt /rootfs/etc/ssl/certs/


FROM scratch

COPY --from=build --chown=10000:10000 /rootfs /

ENV EXPORTER_LISTEN_ADDR="0.0.0.0:9199"
USER 10000:10000
EXPOSE 9199/tcp
ENTRYPOINT ["/bin/graphql_exporter"]
