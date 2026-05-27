FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o marquee ./cmd/server

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/marquee /marquee

ENTRYPOINT ["/marquee"]
