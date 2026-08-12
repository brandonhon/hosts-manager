# Multi-stage build for hosts-manager.
#
# The image is a way to run the CLI without installing Go; it still edits a
# hosts file on the host, which has to be bind-mounted in:
#
#   docker run --rm -v /etc/hosts:/etc/hosts hosts-manager:dev list
#   docker run --rm -v /etc/hosts:/etc/hosts hosts-manager:dev \
#       add 0.0.0.0 ads.example.com -c blocked
#
# Writing that mount needs a uid that owns /etc/hosts on the host, i.e. root.
# Config and backups live under $HOME inside the container and are discarded
# with it unless that is mounted too:
#
#   -v "$HOME/.config/hosts-manager:/root/.config/hosts-manager"
#
# Keep the Go version in step with the go directive in go.mod. It was pinned at
# 1.21 while go.mod required 1.24, so this image had not built at all.
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build

FROM alpine:latest

# ca-certificates only. The previous image installed sudo, which did nothing:
# there was no sudoers entry, and a container that needs more privilege is
# given it by docker run, not by a setuid binary inside the image.
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/build/hosts-manager /usr/local/bin/hosts-manager

# Runs as root deliberately. The tool's job is writing a bind-mounted
# /etc/hosts, which is owned by root on the host, so a non-root user in the
# image cannot do the one thing the image exists for. Container root is not
# host root unless the mount grants it.
WORKDIR /root

# A dry run needs no privileges and touches nothing, so it is a safe default
# and a working health check.
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD hosts-manager --version > /dev/null || exit 1

ENTRYPOINT ["hosts-manager"]
CMD ["--help"]
