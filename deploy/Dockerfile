FROM docker.io/library/golang:1.14-alpine as builder

LABEL maintainer="Jack Murdock <jack_murdock@comcast.com>"

WORKDIR /go/src/github.com/xmidt-org/argus

ARG VERSION=unknown
ARG GITCOMMIT=unknown
ARG BUILDTIME=unknown

ADD https://github.com/geofffranks/spruce/releases/download/v1.25.2/spruce-linux-amd64 /usr/local/bin/spruce
RUN chmod +x /usr/local/bin/spruce

RUN apk add --no-cache --no-progress \
    ca-certificates \
    make \
    git \
    openssh \
    gcc \
    libc-dev \
    upx

COPY . .
RUN make build

FROM alpine:latest

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/xmidt-org/argus/argus.yaml /argus.yaml
COPY --from=builder /go/src/github.com/xmidt-org/argus/argus /argus
COPY --from=builder /go/src/github.com/xmidt-org/argus/deploy/Dockerfile /go/src/github.com/xmidt-org/argus/NOTICE /go/src/github.com/xmidt-org/argus/LICENSE /go/src/github.com/xmidt-org/argus/CHANGELOG.md /
COPY --from=builder /go/src/github.com/xmidt-org/argus/deploy/packaging/entrypoint.sh /entrypoint.sh
COPY --from=builder /go/src/github.com/xmidt-org/argus/deploy/packaging/argus_spruce.yaml /tmp/argus_spruce.yaml
COPY --from=builder /usr/local/bin/spruce /spruce

RUN mkdir /etc/argus/ && touch /etc/argus/argus.yaml && chmod 666 /etc/argus/argus.yaml

USER nobody

ENTRYPOINT ["/entrypoint.sh"]

EXPOSE 6600
EXPOSE 6601
EXPOSE 6602
EXPOSE 6603

CMD ["/argus"]
