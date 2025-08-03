#!/bin/bash

docker pull --platform linux/amd64 tinygo/tinygo-dev:latest

docker run --rm -it \
  -v $PWD:/project \
  -e GOPROXY=https://goproxy.cn,direct \
  tinygo/tinygo-dev:latest \
  /bin/sh -c '
    cd /project && \
    tinygo build -o main.wasm -scheduler=none --no-debug -target=wasip1 .
  '
