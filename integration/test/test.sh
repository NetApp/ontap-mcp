SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SCRIPT_DIR="$(dirname "$SCRIPT_DIR")"
echo "Dir : $SCRIPT_DIR"
cd "$SCRIPT_DIR"/test || exit
export PATH=$PATH:/usr/local/go/bin
if [ -z "$VERSION" ]; then
  VERSION="$(date +%Y.%m.%d%H | cut -c 3-)"
  echo "VERSION not supplied, using $VERSION"
fi

if [ -z "$TEST_PREFIX" ]; then
  TEST_PREFIX="ci"
fi
export TEST_PREFIX
echo "TEST_PREFIX: $TEST_PREFIX"

LD_FLAGS="-X ""'""github.com/netapp/ontap-mcp/version.VERSION=${VERSION}""'"""
echo "$LD_FLAGS"

go mod tidy
go test -v -timeout 1h -ldflags="$LD_FLAGS"