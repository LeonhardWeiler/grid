FROM node:24-alpine AS frontend
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY frontend ./frontend
COPY tsconfig.json vite.config.ts ./
RUN npx tsc --noEmit && npm run build

FROM golang:1.26.2 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY internal ./internal
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -o /grid .

FROM alpine:3.23
WORKDIR /app
COPY --from=builder /grid /app/grid
COPY --from=frontend /app/public/dist ./public/dist
ENV STATE_DIRECTORY=/data STATIC_DIR=/app/public/dist LISTEN_ADDR=0.0.0.0:4000
EXPOSE 4000
STOPSIGNAL SIGTERM
CMD ["/app/grid"]
