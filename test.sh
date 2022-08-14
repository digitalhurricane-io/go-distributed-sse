#!/bin/bash

docker run -p 44293:6379 -d --name sse-redis redis:latest

go test -v -race

docker stop sse-redis
docker rm sse-redis