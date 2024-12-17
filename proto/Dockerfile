FROM golang:1.23.4

ENV PROTOC_VERSION=27.2
ENV PROTOC_GEN_GO_VERSION=1.36.0
ENV PROTOC_GEN_JSONSCHEMA_VERSION=1.4.1

RUN apt-get update && apt-get install unzip

RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
    unzip -o protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local bin/protoc && \
    unzip -o protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local include/* && \
    rm -rf protoc-${PROTOC_VERSION}-linux-x86_64.zip

RUN go install "google.golang.org/protobuf/cmd/protoc-gen-go@v${PROTOC_GEN_GO_VERSION}" && \
    go install "github.com/chrusty/protoc-gen-jsonschema/cmd/protoc-gen-jsonschema@${PROTOC_GEN_JSONSCHEMA_VERSION}"

COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["./entrypoint.sh"]
