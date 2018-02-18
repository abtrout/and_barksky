#!/usr/bin/env bash

docker run --rm \
  -v $PWD:/go/src/github.com/abtrout/and_barksky \
  -w /go/src/github.com/abtrout/and_barksky \
  golang go build
