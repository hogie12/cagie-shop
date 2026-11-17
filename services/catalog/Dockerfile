# Build context is the repository root, because this service imports packages/shared.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY packages/shared packages/shared
COPY services/catalog/go.mod services/catalog/go.sum services/catalog/
WORKDIR /src/services/catalog
RUN go mod download
COPY services/catalog ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/catalog ./cmd/server

FROM alpine:3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/catalog /usr/local/bin/catalog
EXPOSE 8081
ENTRYPOINT ["catalog"]
