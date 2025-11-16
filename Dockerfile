FROM golang:1.25 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/reviewer ./cmd/reviewer

FROM gcr.io/distroless/base-debian12

COPY --from=builder /bin/reviewer /usr/bin/reviewer

EXPOSE 8080

ENTRYPOINT ["/usr/bin/reviewer"]
