#!/bin/bash
if [[ $DOCKER_REGISTRY == "" ]];
  then
    echo "specifiy DOCKER_REGISTRY env"
    exit
  else
    echo "using registry ${DOCKER_REGISTRY}"
fi
TAG=`git rev-parse HEAD`
DATE=`date +%s`
set -x
docker build --ulimit memlock=-1 --ulimit nofile=65535:65535 -t ${DOCKER_REGISTRY}/tgwatch:latest -t ${DOCKER_REGISTRY}/tgwatch:${DATE}_${TAG} . && docker push -a ${DOCKER_REGISTRY}/tgwatch