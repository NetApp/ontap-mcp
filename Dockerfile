ARG GO_VERSION=1.25.6
FROM golang:${GO_VERSION} AS builder

SHELL ["/bin/bash", "-c"]

ARG INSTALL_DIR=/opt/ontap-mcp
ARG BUILD_DIR=/opt/home
ARG VERSION=1.0.0

WORKDIR $BUILD_DIR

RUN mkdir -p $INSTALL_DIR

COPY . .

RUN go build .

RUN cp -a $BUILD_DIR/ontap-mcp $INSTALL_DIR/
RUN mkdir -p $INSTALL_DIR/server/testdata/
RUN cp -a $BUILD_DIR/server/testdata/ontap.yaml $INSTALL_DIR/server/testdata/ 2>/dev/null || true

WORKDIR $BUILD_DIR

ARG INSTALL_DIR=/opt/ontap-mcp

WORKDIR $INSTALL_DIR

ENTRYPOINT ["./ontap-mcp", "start", "--config", "server/testdata/ontap.yaml"]
