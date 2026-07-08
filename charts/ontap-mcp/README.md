# ontap-mcp

A Helm chart for deploying the [ONTAP MCP Server](https://github.com/NetApp/ontap-mcp) on Kubernetes.

## Installing

```bash
helm install my-release ./charts/ontap-mcp
```

This works out of the box with no ONTAP clusters configured yet -- the server
starts but has nothing to poll, and the post-install notes will say so. Provide
`ontapConfig.data` (or `ontapConfig.existingSecret`) via a values file to
configure real clusters -- see "Configuration" below.

## Configuration

The server needs a config file (`ontap.yaml`) describing which ONTAP clusters to
connect to; it's always sourced from a Kubernetes `Secret`, mounted at
`/opt/mcp/ontap.yaml`. Two ways to provide it:

1. **Let the chart create the Secret** by setting `ontapConfig.data` to the full
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

2. **Bring your own Secret** (recommended for anything beyond local testing --
   source it from an external secrets manager via the CSI secrets-store driver
   or the External Secrets Operator instead of putting credentials in a values
   file): create a `Secret` yourself with a key named `ontap.yaml`, then set
   `ontapConfig.existingSecret` to its name. `ontapConfig.data` is ignored when
   this is set.

If neither is set, the server starts with no configured clusters.

### Exposing the service

By default the chart only creates a `ClusterIP` Service. Three ways to expose it,
all off by default:

- `ingress.enabled: true` -- a plain Kubernetes `Ingress`, for clusters with an
  Ingress controller.
- `httpRoute.enabled: true` -- a Gateway API `HTTPRoute` (requires the Gateway
  API installed and a `Gateway` to attach to; set `httpRoute.parentRefs`).
- `listenerSet.enabled: true` -- a Gateway API `ListenerSet` (requires Gateway
  API >= 1.5), for attaching `httpRoute` to a shared per-app listener instead
  of a `Gateway` directly.

### Multiple replicas

The server keeps per-session MCP state in-process. If you scale `replicaCount`
above 1, also add `--stateless` to `extraArgs` (drops the `mcp-session-id`
header requirement) and make sure your Service/Ingress doesn't rely on sticky
sessions.

## Values

| Key | Description | Default |
|---|---|---|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/netapp/ontap-mcp` |
| `image.tag` | Image tag; defaults to the chart's `appVersion` | `""` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `serviceAccount.create` | Create a ServiceAccount | `true` |
| `serviceAccount.automount` | Automount the ServiceAccount token | `false` |
| `securityContext.readOnlyRootFilesystem` | Read-only root filesystem | `true` |
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service/container port; also passed as `--port` | `8080` |
| `extraArgs` | Extra CLI flags, e.g. `--read-only`, `--stateless`, `--json-response` | `[]` |
| `extraEnv` | Extra container env vars | `[]` |
| `ontapConfig.existingSecret` | Name of a pre-existing Secret with an `ontap.yaml` key | `""` |
| `ontapConfig.data` | Full `ontap.yaml` contents; ignored if `existingSecret` is set | `{}` |
| `resources` | Container resource requests/limits | see `values.yaml` |
| `livenessProbe` / `readinessProbe` | Probe definitions against `/health` | see `values.yaml` |
| `extraVolumes` / `extraVolumeMounts` | Additional volumes/mounts | `[]` |
| `ingress.enabled` | Create an Ingress | `false` |
| `httpRoute.enabled` | Create a Gateway API HTTPRoute | `false` |
| `listenerSet.enabled` | Create a Gateway API ListenerSet | `false` |
| `nodeSelector` / `tolerations` / `affinity` | Standard scheduling controls | `{}` / `[]` / `{}` |
