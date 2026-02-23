# GO_VERSION should be overridden by the build script via --build-arg GO_VERSION=$value
ARG GO_VERSION
FROM golang:${GO_VERSION} AS builder

SHELL ["/bin/bash", "-c"]

ARG INSTALL_DIR=/opt/mcp
ARG BUILD_DIR=/opt/home
ARG VERSION=1.0.0
ARG JUST_VERSION=1.42.4
ARG JUST_URL=${JUST_VERSION}/just-${JUST_VERSION}-x86_64-unknown-linux-musl.tar.gz

WORKDIR $BUILD_DIR

RUN mkdir -p $INSTALL_DIR

COPY . .

# Install just, a command runner used in the build process. We use the musl version to ensure it runs in the distroless image.
RUN wget -O - "https://github.com/casey/just/releases/download/${JUST_URL}" \
  | tar -xz -C /usr/local/bin just

RUN VERSION=$VERSION just build

RUN cp -a $BUILD_DIR/ontap-mcp $INSTALL_DIR/

FROM gcr.io/distroless/static-debian12:debug

ARG INSTALL_DIR=/opt/mcp

COPY --from=builder $INSTALL_DIR $INSTALL_DIR

WORKDIR $INSTALL_DIR

ENTRYPOINT ["./ontap-mcp"]
CMD ["start"]

