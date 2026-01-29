# Test Data for ONTAP MCP Tests

This directory contains recorded HTTP request/response files used for testing the ONTAP MCP server without requiring access to a real ONTAP cluster.

## Test Modes

### Replay Mode (Default)
By default, tests run in replay mode using pre-recorded HTTP interactions stored in this directory. This allows tests to run quickly without network access.

```bash
go test ./server/... -run TestApp_CreateVolume
```

### Record Mode
To record new HTTP interactions from a real ONTAP cluster, set the `RECORD_HTTP` environment variable:

```bash
RECORD_HTTP=true go test ./server/... -run TestApp_CreateVolume
```

**Note:** Record mode requires:
1. A valid `ontap.yaml` configuration file in this directory with real cluster credentials
2. Network access to the ONTAP cluster
3. Appropriate permissions to create volumes on the cluster

## Directory Structure

Each test case creates its own subdirectory containing:
- Request files (`.req.txt`): The HTTP request details
- Response files (`.res.txt`): The corresponding HTTP response

For example, the test `create-volume-just-right` will create:
```
testdata/
  create-volume-just-right/
    <hash>.req.txt
    <hash>.res.txt
```

## Updating Test Data

When API responses change or new test cases are added, re-run tests in record mode to update the stored responses.
