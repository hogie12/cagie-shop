# Build context is the repository root, because this service imports packages/shared.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY packages/shared packages/shared
COPY services/order/go.mod services/order/go.sum services/order/
WORKDIR /src/services/order
RUN go mod download
COPY services/order ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/order ./cmd/server

FROM alpine:3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/order /usr/local/bin/order
EXPOSE 8084
ENTRYPOINT ["order"]
