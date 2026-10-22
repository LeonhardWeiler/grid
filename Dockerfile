FROM node:20 AS frontend

WORKDIR /frontend
COPY package.json bun.lockb* ./
RUN npm install -g bun
RUN bun install

COPY frontend .
RUN bun run build


FROM golang:1.26.2 AS builder

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o grid .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/grid /app/grid
COPY --from=frontend /frontend/dist ./public/dist

EXPOSE 4000
CMD ["/app/grid"]
