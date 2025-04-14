#!/bin/bash
docker build -t reviewdog-protoc .
docker run --rm --volume "$(pwd):$(pwd)" --workdir "$(pwd)" reviewdog-protoc
