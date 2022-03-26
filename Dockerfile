FROM docker.io/golang:1 AS build

RUN mkdir -p /rootfs/etc/ssl/certs

RUN echo "nobody:*:10000:nobody" > /rootfs/etc/group \
    && echo "nobody:*:10000:10000:::" > /rootfs/etc/passwd

ARG DEBIAN_FRONTEND="noninteractive"
RUN apt-get update

RUN apt-get install --yes --no-install-recommends \
      make

RUN apt-get install --yes --no-install-recommends \
      ca-certificates \
    && cp -r /etc/ssl/certs/ca-certificates.crt /rootfs/etc/ssl/certs/

COPY . /build
RUN cd /build \
    && make build \
    && mkdir -p /rootfs \
    && cp -r /build/bin /rootfs/


FROM scratch

COPY --from=build --chown=10000:10000 /rootfs /

ENV EXPORTER_LISTEN_ADDR="0.0.0.0:9199"
USER nobody:nobody
WORKDIR /
EXPOSE 9199/TCP
ENTRYPOINT ["/bin/graphql_exporter"]
