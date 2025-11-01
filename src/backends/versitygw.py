import logging
import boto3
import xml.etree.ElementTree as ET

from .backend import Backend
from ..utils import sign_v4_request, send_signed_request

logger = logging.getLogger("s3-resource-operator.versitygw")

class VersityGW(Backend):
    """Backend implementation for VersityGW."""

    def __init__(self, endpoint_url, access_key, secret_key):
        self.endpoint_url = endpoint_url
        self.access_key = access_key
        self.secret_key = secret_key
        self.s3 = boto3.client(
            "s3",
            endpoint_url=endpoint_url,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )

    def create_bucket(self, bucket_name, owner=None):
        if self.bucket_exists(bucket_name):
            logger.info(f"S3 bucket '{bucket_name}' already exists.")
            return
        try:
            self.s3.create_bucket(Bucket=bucket_name)
            logger.info(f"S3 bucket '{bucket_name}' created.")
            if owner:
                self.change_bucket_owner(bucket_name, owner)
        except self.s3.exceptions.BucketAlreadyOwnedByYou:
            logger.info(f"S3 bucket '{bucket_name}' already exists and is owned by you.")
        except self.s3.exceptions.BucketAlreadyExists:
            logger.warning(f"S3 bucket '{bucket_name}' already exists and is owned by someone else.")
        except Exception as e:
            logger.error(f"Error creating bucket '{bucket_name}': {e}")

    def delete_bucket(self, bucket_name):
        try:
            self.s3.delete_bucket(Bucket=bucket_name)
            logger.info(f"S3 bucket '{bucket_name}' deleted.")
        except Exception as e:
            logger.error(f"Error deleting bucket '{bucket_name}': {e}")

    def _list_buckets_raw(self):
        signed_request = sign_v4_request(
            method="PATCH",
            url=f"{self.endpoint_url}/list-buckets",
            region="us-east-1",
            service="s3",
            access_key=self.access_key,
            secret_key=self.secret_key,
            payload_bytes=b"",
        )
        response = send_signed_request(signed_request)
        if response.status_code == 200:
            logger.info("Successfully retrieved bucket list.")
            try:
                root = ET.fromstring(response.content)
                return [
                    {'Name': b.findtext('Name'), 'Owner': b.findtext('Owner')}
                    for b in root.findall('Buckets')
                ]
            except ET.ParseError:
                logger.error("Failed to parse XML response from list-buckets.")
                return []
        else:
            logger.error(f"Error listing buckets: {response.status_code} {response.text}")
            return []

    def bucket_exists(self, bucket_name):
        return any(b['Name'] == bucket_name for b in self._list_buckets_raw())

    def get_bucket_owner(self, bucket_name):
        for bucket in self._list_buckets_raw():
            if bucket.get('Name') == bucket_name:
                return bucket.get('Owner')
        return None

    def change_bucket_owner(self, bucket_name, new_owner):
        url = f"{self.endpoint_url}/change-bucket-owner/?bucket={bucket_name}&owner={new_owner}"
        signed_request = sign_v4_request("PATCH", url, "us-east-1", "s3", self.access_key, self.secret_key, b"")
        response = send_signed_request(signed_request)
        if response.status_code >= 400:
            logger.error(f"Failed to change owner of bucket '{bucket_name}' to '{new_owner}': {response.status_code} {response.text}")
        else:
            logger.info(f"Successfully sent request to change owner of bucket '{bucket_name}' to '{new_owner}'.")

    def create_user(self, access_key, secret_key, role=None, user_id=None, group_id=None):
        if self.user_exists(access_key):
            logger.info(f"IAM user '{access_key}' already exists.")
            return
        root = ET.Element("Account")
        ET.SubElement(root, "Access").text = str(access_key)
        ET.SubElement(root, "Secret").text = str(secret_key)
        ET.SubElement(root, "Role").text = str(role) if role else "user"
        if user_id: ET.SubElement(root, "UserID").text = str(user_id)
        if group_id: ET.SubElement(root, "GroupID").text = str(group_id)
        
        xml_payload = ET.tostring(root)
        signed_request = sign_v4_request("PATCH", f"{self.endpoint_url}/create-user", "us-east-1", "s3", self.access_key, self.secret_key, xml_payload)
        response = send_signed_request(signed_request)
        if response.status_code >= 400:
            logger.error(f"Failed to create user '{access_key}': {response.status_code} {response.text}")
        else:
            logger.info(f"Successfully sent create user request for '{access_key}'.")

    def update_user(self, access_key, secret_key=None, user_id=None, group_id=None):
        root = ET.Element("MutableProps")
        if secret_key: ET.SubElement(root, "Secret").text = str(secret_key)
        if user_id: ET.SubElement(root, "UserID").text = str(user_id)
        if group_id: ET.SubElement(root, "GroupID").text = str(group_id)

        xml_payload = ET.tostring(root)
        url = f"{self.endpoint_url}/update-user?access={access_key}"
        signed_request = sign_v4_request("PATCH", url, "us-east-1", "s3", self.access_key, self.secret_key, xml_payload)
        response = send_signed_request(signed_request)
        if response.status_code >= 400:
            logger.error(f"Failed to update user '{access_key}': {response.status_code} {response.text}")
        else:
            logger.info(f"Successfully sent update request for user '{access_key}'.")

    def delete_user(self, access_key):
        url = f"{self.endpoint_url}/delete-user?access={access_key}"
        signed_request = sign_v4_request("PATCH", url, "us-east-1", "s3", self.access_key, self.secret_key, b"")
        response = send_signed_request(signed_request)
        if response.status_code >= 400:
            logger.error(f"Failed to delete user '{access_key}': {response.status_code} {response.text}")
        else:
            logger.info(f"Successfully deleted user '{access_key}'.")

    def _list_users_raw(self):
        signed_request = sign_v4_request("PATCH", f"{self.endpoint_url}/list-users", "us-east-1", "s3", self.access_key, self.secret_key, b"")
        response = send_signed_request(signed_request)
        if response.status_code == 200:
            try:
                root = ET.fromstring(response.content)
                return [
                    {'Access': a.findtext('Access')}
                    for a in root.findall('Accounts')
                ]
            except ET.ParseError:
                logger.error("Failed to parse XML response from list-users.")
                return []
        else:
            logger.error(f"Failed to list users: {response.status_code} {response.text}")
            return []

    def user_exists(self, access_key):
        return any(u['Access'] == access_key for u in self._list_users_raw())