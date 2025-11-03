import logging
import base64
import hashlib
import botocore.auth
import botocore.awsrequest
import botocore.credentials
import requests

logger = logging.getLogger("s3-resource-operator")


def sign_v4_request(method, url, region, service, access_key, secret_key, payload_bytes, headers=None):
    """
    Sign a request using AWS Signature Version 4.

    :param method: HTTP method (e.g., 'GET', 'POST')
    :param url: The URL to sign
    :param region: AWS region
    :param service: AWS service name
    :param access_key: AWS access key
    :param secret_key: AWS secret key
    :param payload_bytes: The request payload, as bytes
    :param headers: Optional additional headers to include in the signature
    :return: The signed request, with the 'Authorization' header set
    """
    creds = botocore.credentials.Credentials(access_key, secret_key)

    # Create the request object
    aws_request = botocore.awsrequest.AWSRequest(
        method=method.upper(),
        url=url,
        data=payload_bytes,
        headers=headers
    )

    # Calculate the payload hash
    payload_hash = hashlib.sha256(payload_bytes).hexdigest()
    aws_request.headers['X-Amz-Content-Sha256'] = payload_hash

    # Sign the request
    signer = botocore.auth.SigV4Auth(creds, service, region)
    signer.add_auth(aws_request)

    return aws_request


def send_signed_request(signed_request):
    """
    Sends a pre-signed AWSRequest object using the requests library.
    """
    try:
        response = requests.request(
            method=signed_request.method,
            url=signed_request.url,
            headers=signed_request.headers,
            data=signed_request.body
        )
        return response
    except requests.exceptions.RequestException as e:
        logger.error(
            f"Failed to send signed request to {signed_request.url}: {e}")
        # Return a mock response to prevent callers from crashing
        response = requests.Response()
        response.status_code = 500
        response.reason = str(e)
        return response


def get_k8s_api():
    """Initialize and return Kubernetes API client."""
    from kubernetes import client, config

    # Load kube config
    try:
        config.load_incluster_config()
        logger.info("Loaded in-cluster kube config.")
    except config.ConfigException:
        logger.info(
            "Could not load in-cluster config. Falling back to local kube config.")
        config.load_kube_config()
    return client.CoreV1Api()
