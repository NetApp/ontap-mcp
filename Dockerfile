# GO_VERSION should be overridden by the build script via --build-arg GO_VERSION=$value
ARG GO_VERSION
FROM golang:${GO_VERSION} AS builder

SHELL ["/bin/bash", "-c"]

ARG INSTALL_DIR=/opt/mcp
ARG BUILD_DIR=/opt/home
ARG VERSION=1.0.0

WORKDIR $BUILD_DIR

RUN mkdir -p $INSTALL_DIR

COPY . .

RUN GOOS=linux GOARCH=amd64 VERSION=$VERSION go build  .

RUN cp -a $BUILD_DIR/ontap-mcp $INSTALL_DIR/

FROM gcr.io/distroless/static-debian12:debug

ARG INSTALL_DIR=/opt/mcp

COPY --from=builder $INSTALL_DIR $INSTALL_DIR

WORKDIR $INSTALL_DIR

ENTRYPOINT ["./ontap-mcp"]
CMD ["start"]

