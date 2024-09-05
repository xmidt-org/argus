## SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
## SPDX-License-Identifier: Apache-2.0
FROM docker.io/library/golang:1.19-alpine AS builder

WORKDIR /src

RUN apk add --no-cache --no-progress \
    ca-certificates \
    curl

# Download spruce here to eliminate the need for curl in the final image
RUN mkdir -p /go/bin && \
    curl -L -o /go/bin/spruce https://github.com/geofffranks/spruce/releases/download/v1.29.0/spruce-linux-amd64 && \
    chmod +x /go/bin/spruce

COPY . .

##########################
# Build the final image.
##########################

FROM alpine:latest

# Copy over the standard things you'd expect.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt  /etc/ssl/certs/
COPY argus /
COPY .release/docker/entrypoint.sh  /

# Copy over spruce and the spruce template file used to make the actual configuration file.
COPY .release/docker/argus_spruce.yaml  /tmp/argus_spruce.yaml
COPY --from=builder /go/bin/spruce      /bin/

# Include compliance details about the container and what it contains.
COPY Dockerfile /
COPY NOTICE     /
COPY LICENSE    /

# Make the location for the configuration file that will be used.
RUN     mkdir /etc/argus/ \
    &&  touch /etc/argus/argus.yaml \
    &&  chmod 666 /etc/argus/argus.yaml

USER nobody

ENTRYPOINT ["/entrypoint.sh"]

EXPOSE 6600
EXPOSE 6601
EXPOSE 6602
EXPOSE 6603

CMD ["/argus"]
