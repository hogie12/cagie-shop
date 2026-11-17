# Build context is the repository root, because this service imports packages/shared.
# The build stage runs on the builder's own CPU and cross-compiles for the
# target (amd64 or arm64), so multi-arch images need no CPU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY packages/shared packages/shared
COPY services/auth/go.mod services/auth/go.sum services/auth/
WORKDIR /src/services/auth
RUN go mod download
COPY services/auth ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/auth ./cmd/server

FROM alpine:3
COPY --from=build /out/auth /usr/local/bin/auth
# A numeric non-root user needs no RUN step, which keeps this stage emulation-free.
USER 10001:10001
EXPOSE 8082
ENTRYPOINT ["auth"]
