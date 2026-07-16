# Deploy on Kubernetes with Helm

ONTAP-MCP includes an official Helm chart in this repository at `charts/ontap-mcp`.

## Prerequisites

- Kubernetes 1.27+
- Helm 3.12+
- Network connectivity from the ONTAP-MCP pod to your ONTAP clusters

## Installing

```bash
helm install my-release ./charts/ontap-mcp
```

This works out of the box with no ONTAP clusters configured yet. The server
starts but has nothing to poll, and the post-install notes say so. Provide
`ontapConfig.data` or `ontapConfig.existingSecret` via a values file to
configure real clusters.

## Configuration

The server needs a config file, `ontap.yaml`, describing which ONTAP clusters
to connect to. The chart always sources that file from a Kubernetes `Secret`
mounted at `/opt/mcp/ontap.yaml`.

You can provide it in one of two ways:

1. Let the chart create the `Secret` by setting `ontapConfig.data` to the full
   contents of `ontap.yaml`, in its native schema:

   ```yaml
   ontapConfig:
     data:
       Pollers:
         cluster1:
           addr: 10.0.0.1
           username: admin
           password: changeme
           use_insecure_tls: false
   ```

   See [`ontap-example.yaml`](https://github.com/NetApp/ontap-mcp/blob/main/ontap-example.yaml)
   for the complete schema (`Pollers`, `Defaults`, `McpAuth`, `Tls`).

2. Bring your own `Secret`. This is recommended for anything beyond local
   testing. Source it from an external secrets manager through the CSI
   secrets-store driver or the External Secrets Operator instead of putting
   credentials in a values file.

   Create a `Secret` with a key named `ontap.yaml`, then set
   `ontapConfig.existingSecret` to its name. When this is set,
   `ontapConfig.data` is ignored.

If neither option is set, the server still starts with no configured clusters.

## Exposing the service

By default the chart only creates a `ClusterIP` `Service`. Three exposure modes
are available, all disabled by default:

- `ingress.enabled: true` creates a plain Kubernetes `Ingress`.
- `httpRoute.enabled: true` creates a Gateway API `HTTPRoute`. This requires
  the Gateway API to be installed and a `Gateway` to attach to via
  `httpRoute.parentRefs`.
- `listenerSet.enabled: true` creates a Gateway API `ListenerSet`. This
  requires Gateway API 1.5 or newer and is useful when `httpRoute` should
  attach to a shared per-app listener instead of directly to a `Gateway`.

## Multiple replicas

The server keeps per-session MCP state in-process. If you scale
`replicaCount` above `1`, also add `--stateless` to `extraArgs` and make sure
your `Service` or `Ingress` does not rely on sticky sessions.

## Upgrade

```bash
helm upgrade my-release ./charts/ontap-mcp
```

## Uninstall

```bash
helm uninstall my-release
```

## k3d smoke test

Use this to verify the chart behavior in a local k3d cluster:

```bash
k3d cluster create ontap-mcp-smoke
helm lint ./charts/ontap-mcp
helm install ontap-mcp ./charts/ontap-mcp
kubectl rollout status deployment/ontap-mcp --timeout=120s
kubectl port-forward service/ontap-mcp 8080:8080
curl -i http://127.0.0.1:8080/health
```

Expected response:

- HTTP status `200 OK`
- Body `OK`

Delete the local test cluster when done:

```bash
k3d cluster delete ontap-mcp-smoke
```
