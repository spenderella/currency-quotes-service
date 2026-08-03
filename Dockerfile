# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# distroless/static bundles ca-certificates (needed for HTTPS calls to the
# rate provider) and runs as a non-root user by default via the :nonroot tag.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/server /app/server

ENTRYPOINT ["/app/server"]
