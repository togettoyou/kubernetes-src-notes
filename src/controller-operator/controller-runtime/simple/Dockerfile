FROM golang:1.20 AS builder
ENV GO111MODULE=on
ENV GOPROXY https://goproxy.cn,direct
COPY . /app/
WORKDIR /app/
RUN CGO_ENABLED=0 go build -v -o server main.go controller.go

FROM alpine
COPY --from=builder /app/server /app/
WORKDIR /app/
ENTRYPOINT ["./server"]