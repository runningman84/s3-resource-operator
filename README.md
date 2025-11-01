# S3 Resource Operator

[![Build and Publish](https://github.com/<your-github-username>/<your-repo-name>/actions/workflows/publish.yml/badge.svg)](https://github.com/<your-github-username>/<your-repo-name>/actions/workflows/publish.yml)

A Kubernetes operator to manage VersityGW S3 buckets and IAM users declaratively using Kubernetes secrets.

## Overview

The S3 Resource Operator automates the lifecycle of S3 resources within a VersityGW S3-compatible object store. By watching for specially annotated Kubernetes secrets, the operator can automatically:

- Create and delete S3 buckets.
- Create and delete IAM users.
- Assign ownership of a bucket to a newly created user.

This allows for a GitOps-friendly, declarative approach to managing basic S3 resources directly from your Kubernetes manifests.

## Features

- **Automated S3 Resource Provisioning**: Automatically creates S3 buckets and IAM users based on Kubernetes secrets.
- **Backend Support**: Supports multiple S3 backends, including `versitygw`, `minio`, and `garage`.
- **Bucket Ownership Management**: Ensures existing buckets are owned by the correct user, and changes the owner if necessary.
- **Dynamic Reconfiguration**: Watches for changes to secrets and updates resources accordingly.
- **Initial Sync**: On startup, the operator performs a full sync to ensure all declared resources are correctly configured.
- **Configurable**: All settings, including S3 endpoint and credentials, are configurable via environment variables.
- **Helm Chart**: Comes with a Helm chart for easy deployment.

## Prerequisites

- A running Kubernetes cluster (v1.21+).
- [Helm v3+](https://helm.sh/docs/intro/install/) installed.
- A running instance of [VersityGW](https://github.com/versity/versitygw) accessible from the cluster.
- Admin credentials for your VersityGW instance.

## Installation

The operator is deployed using a Helm chart.

1.  **Add the Helm Repository**

    First, add the Helm repository that is published via GitHub Pages.

    ```sh
    helm repo add <your-repo-name> https://<your-github-username>.github.io/<your-repo-name>
    helm repo update
    ```

2.  **Install the Chart**

    Install the chart into your cluster. You must provide the admin credentials and the endpoint URL for your VersityGW instance.

    ```sh
helm install s3-resource-operator <your-repo-name>/s3-resource-operator \\
      --namespace s3-resource-operator \\
      --create-namespace \
      --set versitygw.endpointUrl="http://<your-versitygw-service>:7070" \
      --set versitygw.adminAccessKey="<your-admin-access-key>" \
      --set versitygw.adminSecretKey="<your-admin-secret-key>"
    ```

## Usage

To provision a new bucket and user, create a Kubernetes `Secret` with the required annotation and data fields.

The operator looks for secrets with the annotation `s3-resource-operator/enabled: "true"`.

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
    s3-resource-operator/enabled: "true"
type: Opaque
stringData:
  # Required fields
  bucket-name: "my-app-backups"
  access-key: "my-app-user"
  access-secret: "a-very-strong-and-long-password"

  # Optional fields
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

When you delete the secret, the operator will automatically delete the user and the bucket.

The secret should contain the following data fields:

- `bucket-name`: The name of the S3 bucket. **(Required)**
- `access-key`: The access key for the IAM user. **(Required)**
- `access-secret`: The secret key for the IAM user. **(Required)**
- `role`: (Optional) The role to assign to the user.
- `user-id`: (Optional) The user ID to assign to the user.
- `group-id`: (Optional) The group ID to assign to the user.

### External Secrets

It is recommended to use a tool like the [External Secrets Operator](https://external-secrets.io/) to manage the secrets that this operator consumes. This allows you to store your S3 credentials in a secure secret store like Vault, AWS Secrets Manager, or Google Secrets Manager. An example of an `ExternalSecret` can be found in `crontrib/example-external-secret.yaml`.

## Development

To run the operator locally for development, you need Python 3.9+ and the required dependencies.

1.  **Clone the repository:**

    ```sh
    git clone https://github.com/<your-github-username>/<your-repo-name>.git
    cd <your-repo-name>
    ```

2.  **Install dependencies:**

    ```sh
    pip install -r requirements.txt
    ```

3.  **Set Environment Variables:**

    The operator reads its configuration from environment variables.

    ```sh
    export KUBECONFIG=~/.kube/config
    export S3_ENDPOINT_URL="http://<your-versitygw-endpoint>"
    export S3_ACCESS_KEY="<your-admin-access-key>"
    export S3_SECRET_KEY="<your-admin-secret-key>"
    export ANNOTATION_KEY="s3-resource-operator/enabled"
    ```

4.  **Run the operator:**

    ```sh
    python src/main.py
    ```

## CI/CD

This project uses GitHub Actions for continuous integration and deployment. The workflow in `.github/workflows/publish.yml` handles:
- **Building the Docker Image**: On every push to `main`, the Docker image is built and pushed to the GitHub Container Registry (GHCR).
- **Publishing the Helm Chart**: The Helm chart is packaged and published to a `gh-pages` branch, which serves as a public Helm repository.

## Configuration

The operator can be configured using the following environment variables:

| Environment Variable      | Description                                                                 | Default                        |
| ------------------------- | --------------------------------------------------------------------------- | ------------------------------ |
| `ANNOTATION_KEY`          | The annotation key to look for on secrets.                                  | `s3-resource-operator/enabled` |
| `S3_ENDPOINT_URL`         | The URL of the S3 endpoint.                                                 | (required)                     |
| `S3_ACCESS_KEY`           | The access key for the S3 endpoint (for the operator itself).               | (required)                     |
| `S3_SECRET_KEY`           | The secret key for the S3 endpoint (for the operator itself).               | (required)                     |
| `BACKEND_NAME`            | The name of the S3 backend to use (`versitygw`, `minio`, `garage`).         | `versitygw`                    |

## Contributing

Contributions are welcome! Please feel free to submit a pull request.

This project uses [Renovate](https://github.com/renovatebot/renovate) to keep dependencies up-to-date. Renovate will automatically create pull requests for outdated dependencies.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.