#!/bin/bash
set -x
docker build --ulimit memlock=-1 --ulimit nofile=65535:65535 -t bee:5001/tgwatch:latest . && docker push bee:5001/tgwatch:latest