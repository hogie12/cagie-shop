# Build context is the repository root, because this service imports packages/shared.
# The build stage runs on the builder's own CPU and cross-compiles for the
# target (amd64 or arm64), so multi-arch images need no CPU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY packages/shared packages/shared
COPY services/payment/go.mod services/payment/go.sum services/payment/
WORKDIR /src/services/payment
RUN go mod download
COPY services/payment ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/payment ./cmd/server

FROM alpine:3
COPY --from=build /out/payment /usr/local/bin/payment
# A numeric non-root user needs no RUN step, which keeps this stage emulation-free.
USER 10001:10001
EXPOSE 8085
ENTRYPOINT ["payment"]
