FROM golang:1.25-alpine AS builder

WORKDIR /app

# go.mod と go.sum を先にコピーして依存関係をキャッシュ
COPY go.mod go.sum ./
RUN go mod download

# 全ソースコードをコピー
COPY . .

# 全ての .go ファイルをコンパイル対象に含める
RUN go build -o /omron_batch batch/*.go

FROM alpine:latest
RUN apk --no-cache add tzdata
ENV TZ=Asia/Tokyo

COPY --from=builder /omron_batch /omron_batch
CMD ["/omron_batch"]
