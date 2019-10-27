#bin/bash -xe

docker-compose down
docker build -t redis-proxy:1.0 . || exit 1
docker-compose up --abort-on-container-exit