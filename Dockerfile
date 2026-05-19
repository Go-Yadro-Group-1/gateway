FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/jira-gateway ./cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/jira-gateway ./jira-gateway
COPY config ./config

ENV PORT=8080
EXPOSE ${PORT}

ENTRYPOINT ["./jira-gateway"]
CMD ["serve", "--config", "config/dev.yaml"]
