# syntax=docker/dockerfile:1.7

FROM gcr.io/distroless/base-debian12:nonroot

ARG VERSION
COPY --chmod=755 build/gateway/${VERSION}/gateway /gateway

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/gateway"]
