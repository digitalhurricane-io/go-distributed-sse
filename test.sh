#!/bin/bash

docker run -P 6379:6379 --name sse-redis redis:latest

go test -v

docker stop sse-redis
docker rm sse-redis