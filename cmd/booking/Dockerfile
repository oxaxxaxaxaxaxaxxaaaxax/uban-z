# syntax=docker/dockerfile:1.7

# ---- build stage ----
FROM golang:1.25.7-alpine AS build

WORKDIR /src

# Cache module downloads as a separate layer.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source.
COPY cmd ./cmd
COPY internal ./internal

# Static binary, no debug symbols, reproducible paths.
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/booking \
        ./cmd/booking

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/booking /booking

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/booking"]
