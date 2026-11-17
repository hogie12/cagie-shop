# Build context is the repository root, because this service imports packages/shared.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY packages/shared packages/shared
COPY services/auth/go.mod services/auth/go.sum services/auth/
WORKDIR /src/services/auth
RUN go mod download
COPY services/auth ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/auth ./cmd/server

FROM alpine:3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/auth /usr/local/bin/auth
EXPOSE 8082
ENTRYPOINT ["auth"]
