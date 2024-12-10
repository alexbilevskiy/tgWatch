#!/bin/bash
if [[ $DOCKER_REGISTRY == "" ]];
  then
    echo "specifiy DOCKER_REGISTRY env"
    exit
  else
    echo "using registry ${DOCKER_REGISTRY}"
fi
set -x
docker build --ulimit memlock=-1 --ulimit nofile=65535:65535 --build-context local-src=/opt/src -t ${DOCKER_REGISTRY}/tgwatch:latest . && docker push ${DOCKER_REGISTRY}/tgwatch:latest