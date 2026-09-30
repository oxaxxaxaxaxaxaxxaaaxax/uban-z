# syntax=docker/dockerfile:1.7

FROM gcr.io/distroless/base-debian12:nonroot

ARG VERSION
COPY --chmod=755 build/booking/${VERSION}/booking /booking

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/booking"]
