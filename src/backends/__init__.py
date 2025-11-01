from .versitygw import VersityGW
from .garage import Garage
from .minio import Minio

SUPPORTED_BACKENDS = {
    "versitygw": VersityGW,
    "garage": Garage,
    "minio": Minio,
}

def get_backend(name, endpoint_url, access_key, secret_key):
    backend_class = SUPPORTED_BACKENDS.get(name.lower())
    if not backend_class:
        raise ValueError(f"Unsupported backend: {name}")
    return backend_class(endpoint_url, access_key, secret_key)