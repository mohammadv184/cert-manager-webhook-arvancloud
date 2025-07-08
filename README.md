# cert-manager-webhook-arvancloud

A [cert-manager](https://cert-manager.io) dns01 challenge solver for [Arvancloud](https://www.arvancloud.ir).

This webhook enables you to use ArvanCloud DNS for DNS01 challenge validation when issuing certificates with cert-manager.

## Installation

### Prerequisites

- Kubernetes cluster
- [cert-manager](https://cert-manager.io/docs/installation/) installed
- ArvanCloud account with API access

### Installing the Webhook


## Configuration

### Creating a Secret with ArvanCloud API Key

Create a Secret in the cert-manager namespace containing your ArvanCloud API key:

```bash
kubectl create secret generic arvancloud-api-key \
  --namespace cert-manager \
  --from-literal=api-key="your-arvancloud-api-key-here"
```

### Creating an Issuer/ClusterIssuer

Create an Issuer or ClusterIssuer resource that uses the ArvanCloud webhook for DNS01 validation:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging-arvancloud
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: user@example.com
    privateKeySecretRef:
      name: letsencrypt-staging-arvancloud
    solvers:
    - dns01:
        webhook:
          groupName: acme.arvancloud.ir
          solverName: arvancloud
          config:
            apiKeySecretRef:
              name: arvancloud-api-key
              key: api-key
            # Optional TTL in seconds for DNS records (default: 120)
            ttl: 120
```

For a production environment, use the production Let's Encrypt server:
```
server: https://acme-v02.api.letsencrypt.org/directory
```

### Creating a Certificate

Create a Certificate resource that uses the Issuer/ClusterIssuer:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-com
  namespace: default
spec:
  secretName: example-com-tls
  issuerRef:
    name: letsencrypt-staging-arvancloud
    kind: ClusterIssuer
  dnsNames:
  - example.com
  - "*.example.com"
```

## Troubleshooting

Check the logs of the webhook pod:
```bash
kubectl logs -n cert-manager -l app=cert-manager-webhook-arvancloud
```

Check the status of the Certificate:
```bash
kubectl describe certificate example-com
```

## Development

### Building the Webhook Locally

```bash
go mod tidy
go build -o webhook main.go arvancloud.go
```

### Running Tests

```bash
go test ./...
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.