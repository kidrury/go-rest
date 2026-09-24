FROM golang:1.27.1-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/rest-pro \
    ./cmd/api

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /out/rest-pro /rest-pro

EXPOSE 8099

USER nonroot:nonroot

ENTRYPOINT ["/rest-pro"]