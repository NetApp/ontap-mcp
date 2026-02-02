ARG GO_VERSION=1.25.6
FROM golang:${GO_VERSION} AS builder

SHELL ["/bin/bash", "-c"]

ARG INSTALL_DIR=/opt/mcp
ARG BUILD_DIR=/opt/home
ARG VERSION=1.0.0

WORKDIR $BUILD_DIR

RUN mkdir -p $INSTALL_DIR

COPY . .

RUN ls -R $INSTALL_DIR
RUN ls -R $BUILD_DIR

RUN GOOS=linux GOARCH=amd64 VERSION=$VERSION go build  .

RUN cp -a $BUILD_DIR/ontap-mcp $INSTALL_DIR/
RUN mkdir -p $INSTALL_DIR/server/testdata/
RUN cp -a $BUILD_DIR/server/testdata/ontap.yaml $INSTALL_DIR/server/testdata/ 2>/dev/null || true

FROM gcr.io/distroless/static-debian12:debug

ARG INSTALL_DIR=/opt/mcp

COPY --from=builder $INSTALL_DIR $INSTALL_DIR

WORKDIR $INSTALL_DIR

ENTRYPOINT ["./ontap-mcp", "start", "--config", "server/testdata/ontap.yaml"]
