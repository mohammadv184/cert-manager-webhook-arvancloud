<p align="center" style="display:flex;align-items:center;justify-content:center;gap:16px;">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" alt="cert-manager" height="60" style="border-radius:50%;background:#fff;"/>
  <span style="display:inline-block;height:60px;vertical-align:middle;"><svg height="60" width="2"><rect width="2" height="60" style="fill:#888;"/></svg></span>
  <img src="https://mohammad-abbasi.me/arvan-logo.png" alt="ArvanCloud" height="60" style="border-radius:50%;background:#fff;"/>
</p>

# cert-manager-webhook-arvancloud

A [cert-manager](https://cert-manager.io) dns01 challenge solver for [Arvancloud](https://www.arvancloud.ir).

This webhook enables you to use Arvancloud DNS for DNS01 challenge validation when issuing certificates with cert-manager.

## Installation

### Prerequisites

- Kubernetes cluster
- [cert-manager](https://cert-manager.io/docs/installation/) installed
- [Arvancloud APIKey](https://docs.arvancloud.ir/en/accounts/iam/machine-user) with [Access Policy](https://docs.arvancloud.ir/en/accounts/iam/policies) to the required domains

### Installing the Webhook

You can deploy the webhook using the provided Helm chart:

```bash
helm repo add arvancloud-webhook https://mohammad-abbasi.me/cert-manager-webhook-arvancloud
helm install cert-manager-webhook-arvancloud arvancloud-webhook/cert-manager-webhook-arvancloud -n cert-manager
```

Or deploy manually using the manifests in the `charts/` directory.

## Configuration

### Creating a Secret with Arvancloud API Key

Create a Secret in the cert-manager namespace containing your Arvancloud API key:

```bash
kubectl create secret generic arvancloud-api-key \
  --namespace cert-manager \
  --from-literal=apikey="your-arvancloud-api-key-here"
```

Or you can create the Secret using a YAML manifest:

```yaml
apiVersion: v1
kind: Secret
metadata:
    name: arvancloud-api-key
    namespace: cert-manager
stringData:
  apikey: "apikey xxxxxxxx.xxxxxxxxxxx.xxxxxx"
```

### Creating an Issuer/ClusterIssuer

Create an Issuer or ClusterIssuer resource that uses the Arvancloud webhook for DNS01 validation:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: arvancloud-issuer
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: user@example.com
    privateKeySecretRef:
      name: arvancloud-issuer-key
    solvers:
    - dns01:
        webhook:
          groupName: acme.arvancloud.ir
          solverName: arvancloud
          config:
            apiKeySecretRef:
              name: arvancloud-api-key
              key: apikey
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
    name: arvancloud-issuer
    kind: ClusterIssuer
  dnsNames:
  - example.com
  - "*.example.com"
```

## Troubleshooting

- Check the logs of the webhook pod:

```bash
kubectl logs -n cert-manager -l app=cert-manager-webhook-arvancloud
```

- Check the status of the Certificate:

```bash
kubectl describe certificate example-com
```

- Ensure your API key has the correct permissions for DNS management in Arvancloud.


## Contributing

Contributions are welcome! Please open issues or pull requests for improvements or bug fixes.

## Security

If you discover any security-related issues, please email mohammadv184@gmail.com instead of using the issue tracker.

## Credits

- [Mohammad Abbasi](https://mohammad-abbasi.me) - Author and Maintainer
- [All Contributors](../../contributors)


## License

The MIT License (MIT). Please see [License File](LICENSE) for more information.