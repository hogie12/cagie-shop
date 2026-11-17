# Build context is the repository root, because this service imports packages/shared.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY packages/shared packages/shared
COPY services/inventory/go.mod services/inventory/go.sum services/inventory/
WORKDIR /src/services/inventory
RUN go mod download
COPY services/inventory ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/inventory ./cmd/server

FROM alpine:3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/inventory /usr/local/bin/inventory
EXPOSE 8083
ENTRYPOINT ["inventory"]
