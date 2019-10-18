FROM golang

WORKDIR redis-proxy
COPY . .

RUN go build
CMD "./redis-proxy"