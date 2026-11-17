# Build context is the repository root, because this service imports packages/shared.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY packages/shared packages/shared
COPY services/gateway/go.mod services/gateway/go.sum services/gateway/
WORKDIR /src/services/gateway
RUN go mod download
COPY services/gateway ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/server

FROM alpine:3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/gateway /usr/local/bin/gateway
EXPOSE 8080
ENTRYPOINT ["gateway"]
