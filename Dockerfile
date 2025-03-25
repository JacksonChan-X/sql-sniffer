FROM --platform=linux/amd64 alpine:latest

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk add --no-cache go build-base libpcap-dev musl-dev musl-utils gcc g++ gcompat

WORKDIR /app

COPY . .

RUN go env -w GOPROXY=https://goproxy.cn,direct

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC="gcc" go build -o sql-sniffer -tags netgo -ldflags "-extldflags '-static'"
