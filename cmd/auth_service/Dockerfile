# syntax=docker/dockerfile:1.7

FROM golang:1.25.7-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/auth-service \
        ./cmd/auth_service

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/auth-service /auth-service

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/auth-service"]
