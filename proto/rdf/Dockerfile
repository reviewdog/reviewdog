FROM golang:1.17.0

ENV PROTOC_VERSION=3.12.3

RUN apt-get update && apt-get install unzip

RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
    unzip -o protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local bin/protoc && \
    unzip -o protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local include/* && \
    rm -rf protoc-${PROTOC_VERSION}-linux-x86_64.zip

RUN go get -u google.golang.org/protobuf/cmd/protoc-gen-go && \
    go get github.com/chrusty/protoc-gen-jsonschema/cmd/protoc-gen-jsonschema

RUN mkdir /workdir
WORKDIR /workdir

COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
