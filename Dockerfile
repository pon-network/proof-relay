# syntax=docker/dockerfile:1
FROM golang:1.20 as builder
ARG VERSION
WORKDIR /build

# Cache for the modules
COPY go.mod ./
COPY go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download

# Now adding all the code and start building
ADD . .
RUN --mount=type=cache,target=/root/.cache/go-build GOOS=linux go build -trimpath -ldflags "-s -X cmd.RelayVersion=$VERSION -X main.RelayVersion=$VERSION -linkmode external -extldflags '-static'" -v -o pon-relay .

FROM alpine
RUN apk add --no-cache libstdc++ libc6-compat
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/pon-relay /app/pon-relay
ENTRYPOINT ["/app/pon-relay"]