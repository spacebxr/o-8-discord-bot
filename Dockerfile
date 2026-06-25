FROM node:20-alpine AS dashboard-builder
WORKDIR /app/dashboard
COPY dashboard/package*.json ./
RUN npm install
COPY dashboard/ ./
RUN npm run build

FROM golang:1.26-alpine AS go-builder
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o bot cmd/bot/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates ffmpeg opus
COPY --from=go-builder /app/bot /bot
COPY --from=dashboard-builder /app/dashboard/dist /dashboard/dist
ENTRYPOINT ["/bot"]
