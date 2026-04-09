#!/bin/bash
if [[ $DOCKER_REGISTRY == "" ]];
  then
    echo "specifiy DOCKER_REGISTRY env"
    exit
  else
    echo "using registry ${DOCKER_REGISTRY}"
fi
set -x

docker build -f Dockerfile.tdlib --ulimit memlock=-1 --ulimit nofile=65535:65535 -t ${DOCKER_REGISTRY}/tdlib:latest . && \
docker push ${DOCKER_REGISTRY}/tdlib:latest

docker build --build-arg DOCKER_REGISTRY=${DOCKER_REGISTRY} -f Dockerfile -t ${DOCKER_REGISTRY}/tgwatch:latest . && \
docker push -a ${DOCKER_REGISTRY}/tgwatch