# S3 Resource Operator

[![Build and Publish](https://github.com/runningman84/s3-resource-operator/actions/workflows/publish.yml/badge.svg)](https://github.com/runningman84/s3-resource-operator/actions/workflows/publish.yml)

A Kubernetes operator to manage S3 buckets and IAM users for S3-compatible object stores like VersityGW, MinIO, and Garage, declaratively using Kubernetes secrets.

## Overview

The S3 Resource Operator automates the lifecycle of S3 resources within a compatible object store (e.g., VersityGW, MinIO, Garage). By watching for specially annotated Kubernetes secrets, the operator can automatically:

- Create S3 buckets.
- Create IAM users.
- Assign ownership of a bucket to a newly created user.

This allows for a GitOps-friendly, declarative approach to managing basic S3 resources directly from your Kubernetes manifests.

## Features

- **Automated S3 Resource Provisioning**: Automatically creates S3 buckets and IAM users based on Kubernetes secrets.
- **Backend Support**: Supports multiple S3 backends, including `versitygw`, `minio`, and `garage`.
- **Bucket Ownership Management**: Ensures existing buckets are owned by the correct user, and changes the owner if necessary.
- **Dynamic Reconfiguration**: Watches for changes to secrets and updates resources accordingly.
- **Initial Sync**: On startup, the operator performs a full sync to ensure all declared resources are correctly configured.
- **Graceful Shutdown**: Properly handles SIGTERM and SIGINT signals for clean shutdown in Kubernetes environments.
- **Prometheus Metrics**: Exposes metrics on port 8000 for monitoring (secrets processed, errors, sync duration).
- **Configurable**: All settings, including S3 endpoint and credentials, are configurable via environment variables.
- **Helm Chart**: Comes with a Helm chart for easy deployment via OCI registry.

## Prerequisites

- A running Kubernetes cluster (v1.21+).
- [Helm v3+](https://helm.sh/docs/intro/install/) installed.
- A running instance of an S3-compatible object store (e.g., [VersityGW](https://github.com/versity/versitygw), [MinIO](https://min.io/)) accessible from the cluster.
- Admin credentials for your S3 instance.

## Installation

The operator is deployed using a Helm chart published to GitHub Container Registry (OCI).

1.  **Install from OCI Registry**

    Install the chart directly from the OCI registry:

    ```sh
    helm install s3-resource-operator oci://ghcr.io/runningman84/s3-resource-operator \
          --version 0.1.0 \
          --namespace s3-resource-operator \
          --create-namespace \
          --set operator.secret.data.S3_ENDPOINT_URL="http://<your-s3-service-endpoint>" \
          --set operator.secret.data.S3_ACCESS_KEY="<your-admin-access-key>" \
          --set operator.secret.data.S3_SECRET_KEY="<your-admin-secret-key>"
    ```

2.  **Alternative: Install from local chart**

    Or clone the repository and install locally:

    ```sh
    git clone https://github.com/runningman84/s3-resource-operator.git
    cd s3-resource-operator
    helm install s3-resource-operator ./helm \
          --namespace s3-resource-operator \
          --create-namespace \
          --set operator.secret.data.S3_ENDPOINT_URL="http://<your-s3-service-endpoint>" \
          --set operator.secret.data.S3_ACCESS_KEY="<your-admin-access-key>" \
          --set operator.secret.data.S3_SECRET_KEY="<your-admin-secret-key>"
    ```

## Usage

To provision a new bucket and user, create a Kubernetes `Secret` with the required annotation and data fields.

The operator looks for secrets with the annotation `s3-resource-operator.io/enabled: "true"`.

### Example Secret

Here is an example of a secret that will trigger the operator to create a bucket named `my-app-backups` and a user with the access key `my-app-user`.

Create a file named `my-app-secret.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-app-s3-credentials
  namespace: my-app
  annotations:
    s3-resource-operator.io/enabled: "true"
type: Opaque
stringData:
  # Required fields
  bucket-name: "my-app-backups"
  access-key: "my-app-user"
  access-secret: "a-very-strong-and-long-password"

  # Optional fields
  endpoint-url: "http://<your-s3-service-endpoint>"
  role: "user"
  user-id: "1001"
  group-id: "1001"
```

Apply it to your cluster:

```sh
kubectl apply -f my-app-secret.yaml
```

Once applied, the operator will perform the following actions:
1.  Create an IAM user named `my-app-user`.
2.  Create an S3 bucket named `my-app-backups`.
3.  Change the owner of the `my-app-backups` bucket to `my-app-user`.

When you delete the secret, the operator will not delete the user or the bucket. This is a safety measure to prevent accidental data loss.

The secret should contain the following data fields:

- `bucket-name`: The name of the S3 bucket. **(Required)**
- `access-key`: The access key for the IAM user. **(Required)**
- `access-secret`: The secret key for the IAM user. **(Required)**
- `endpoint-url`: (Optional) The S3 endpoint URL. If provided, it must match the operator's `S3_ENDPOINT_URL` configuration.
- `role`: (Optional) The role to assign to the user.
- `user-id`: (Optional) The user ID to assign to the user.
- `group-id`: (Optional) The group ID to assign to the user.

### External Secrets

It is recommended to use a tool like the [External Secrets Operator](https://external-secrets.io/) to manage the secrets that this operator consumes. This allows you to store your S3 credentials in a secure secret store like Vault, AWS Secrets Manager, or Google Secrets Manager. An example of an `ExternalSecret` can be found in `crontrib/example-external-secret.yaml`.

## Development

To run the operator locally for development, you need Python 3.12+ and the required dependencies.

1.  **Clone the repository:**

    ```sh
    git clone https://github.com/runningman84/s3-resource-operator.git
    cd s3-resource-operator
    ```

2.  **Install dependencies:**

    ```sh
    pip install -r requirements.txt
    ```

3.  **Set Environment Variables:**

    The operator reads its configuration from environment variables.

    ```sh
    export KUBECONFIG=~/.kube/config
    export S3_ENDPOINT_URL="http://<your-s3-endpoint>"
    export S3_ACCESS_KEY="<your-admin-access-key>"
    export S3_SECRET_KEY="<your-admin-secret-key>"
    export ANNOTATION_KEY="s3-resource-operator.io/enabled"
    export BACKEND_NAME="versitygw"
    ```

4.  **Run the operator:**

    ```sh
    python3 -m src.main
    ```

## CI/CD

This project uses GitHub Actions for continuous integration and deployment. The workflows handle:
- **Testing**: Runs tests on every push and pull request (`.github/workflows/test.yml`)
- **Release**: Creates semantic releases and tags when code is merged to main (`.github/workflows/release.yml`)
- **Publishing**: Builds and publishes Docker images and Helm charts to GHCR when tags are created (`.github/workflows/publish.yml`)

### Release Flow
1. Code is merged to `main` branch
2. Semantic-release creates a new version tag
3. Docker image is built and pushed to `ghcr.io/runningman84/s3-resource-operator`
4. Helm chart is packaged and pushed to `oci://ghcr.io/runningman84/s3-resource-operator`

## Configuration

The operator can be configured using the following environment variables:

| Environment Variable      | Description                                                                 | Default                        |
| ------------------------- | --------------------------------------------------------------------------- | ------------------------------ |
| `ANNOTATION_KEY`          | The annotation key to look for on secrets.                                  | `s3-resource-operator.io/enabled` |
| `S3_ENDPOINT_URL`         | The URL of the S3 endpoint.                                                 | (required)                     |
| `S3_ACCESS_KEY`           | The access key for the S3 endpoint (for the operator itself).               | (required)                     |
| `S3_SECRET_KEY`           | The secret key for the S3 endpoint (for the operator itself).               | (required)                     |
| `BACKEND_NAME`            | The name of the S3 backend to use (`versitygw`, `minio`, `garage`).         | `versitygw`                    |

## Monitoring

The operator exposes Prometheus metrics and a health check endpoint on port 8000:

### Health Check
- **Endpoint**: `/healthz`
- **Purpose**: Kubernetes liveness and readiness probes
- **Response**: `{"status":"healthy"}`

```sh
kubectl port-forward -n s3-resource-operator deployment/s3-resource-operator 8000:8000
curl http://localhost:8000/healthz
```

### Metrics
- **Endpoint**: `/metrics`
- **Available Metrics**:
  - `s3_operator_secrets_processed_total`: Total number of secrets processed
  - `s3_operator_errors_total`: Total number of errors encountered
  - `s3_operator_sync_duration_seconds`: Duration of sync cycles (histogram)
  - `s3_operator_handle_secret_duration_seconds`: Duration of handling individual secrets (histogram)

To access metrics:
```sh
kubectl port-forward -n s3-resource-operator deployment/s3-resource-operator 8000:8000
curl http://localhost:8000/metrics
```

### Prometheus Operator
If you're using the Prometheus Operator, you can enable automatic service discovery by setting `serviceMonitor.enabled=true` in your Helm values:

```yaml
serviceMonitor:
  enabled: true
  interval: 30s
```

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on our development process, coding standards, and how to submit pull requests.

- **[Contributing Guide](CONTRIBUTING.md)** - Development setup and contribution guidelines
- **[Code of Conduct](CODE_OF_CONDUCT.md)** - Community standards and expectations
- **[Security Policy](SECURITY.md)** - How to report security vulnerabilities

This project uses [Renovate](https://github.com/renovatebot/renovate) to keep dependencies up-to-date. Renovate will automatically create pull requests for outdated dependencies.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.