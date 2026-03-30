# Stage 1: 프론트엔드 빌드
# vite.config.ts의 outDir이 ../web/static이므로 결과물이 /app/web/static에 출력된다.
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Go 바이너리 빌드
# go-sqlite3가 CGO로 컴파일되므로 CGO_ENABLED=1과 gcc가 포함된 bookworm 이미지를 사용한다.
FROM golang:1.25-bookworm AS go-builder
WORKDIR /app
ENV CGO_ENABLED=1
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Stage 1 결과물을 embed 대상 경로에 복사한 뒤 go build하면 staticFS에 자동 포함된다.
COPY --from=frontend-builder /app/web/static ./web/static
RUN go build -ldflags="-s -w" -o server ./backend/cmd/server

# Stage 3: 런타임
# Litestream이 libc를 필요로 하고 ca-certificates는 오브젝트 스토리지 HTTPS 연결에 필요하다.
FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=litestream/litestream:latest /usr/local/bin/litestream /usr/local/bin/litestream
WORKDIR /app
COPY --from=go-builder /app/server .
COPY litestream.yml /etc/litestream.yml
RUN mkdir -p /data
EXPOSE 8080
# Litestream이 앱 프로세스를 자식으로 실행해 WAL 복제와 앱 수명을 함께 관리한다.
CMD ["litestream", "replicate", "-exec", "./server", "-config", "/etc/litestream.yml"]
