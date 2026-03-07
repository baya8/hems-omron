FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod ./
# COPY go.sum* ./
RUN go mod download

COPY . .
RUN go build -o /omron_batch batch/main.go batch/db.go batch/omron.go batch/models.go

FROM alpine:latest
RUN apk --no-cache add tzdata
ENV TZ=Asia/Tokyo

COPY --from=builder /omron_batch /omron_batch
CMD ["/omron_batch"]
