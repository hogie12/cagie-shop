# Build context is the repository root, because this service imports packages/shared.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY packages/shared packages/shared
COPY services/payment/go.mod services/payment/go.sum services/payment/
WORKDIR /src/services/payment
RUN go mod download
COPY services/payment ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/payment ./cmd/server

FROM alpine:3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/payment /usr/local/bin/payment
EXPOSE 8085
ENTRYPOINT ["payment"]
