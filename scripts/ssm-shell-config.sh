#!/bin/bash
set -e

export AWS_PAGER=""

ACTION=${1:-setup}
REGION=${2:-$(aws configure get region)}

if [ -z "$REGION" ]; then
  echo "Error: No region specified and no default region configured"
  echo "Usage: $0 [setup|delete] [region]"
  exit 1
fi

if [ "$ACTION" = "delete" ]; then
  echo "Deleting SSM Session Manager document for region: $REGION"
  if aws ssm describe-document --name "SSM-SessionManagerRunShell" --region "$REGION" &>/dev/null; then
    aws ssm delete-document --name "SSM-SessionManagerRunShell" --region "$REGION"
    echo "✓ Document deleted from $REGION"
  else
    echo "Document does not exist in $REGION"
  fi
  exit 0
fi

echo "Configuring SSM Session Manager for region: $REGION"

# Check if document exists
if aws ssm describe-document --name "SSM-SessionManagerRunShell" --region "$REGION" &>/dev/null; then
  echo "Document exists, updating..."
  aws ssm update-document \
    --name "SSM-SessionManagerRunShell" \
    --content '{
      "schemaVersion": "1.0",
      "description": "Auto switch to UID 1000 user if exists",
      "sessionType": "Standard_Stream",
      "inputs": {
        "s3BucketName": "",
        "s3KeyPrefix": "",
        "s3EncryptionEnabled": false,
        "cloudWatchLogGroupName": "",
        "cloudWatchEncryptionEnabled": false,
        "kmsKeyId": "",
        "runAsEnabled": false,
        "runAsDefaultUser": "",
        "shellProfile": {
          "windows": "",
          "linux": "user=$(id -un 1000 2>/dev/null); [ -n \"$user\" ] && sudo su - $user || /bin/bash"
        }
      }
    }' \
    --document-version '$LATEST' \
    --region "$REGION"
else
  echo "Document does not exist, creating..."
  aws ssm create-document \
    --name "SSM-SessionManagerRunShell" \
    --document-type "Session" \
    --content '{
      "schemaVersion": "1.0",
      "description": "Auto switch to UID 1000 user if exists",
      "sessionType": "Standard_Stream",
      "inputs": {
        "s3BucketName": "",
        "s3KeyPrefix": "",
        "s3EncryptionEnabled": false,
        "cloudWatchLogGroupName": "",
        "cloudWatchEncryptionEnabled": false,
        "kmsKeyId": "",
        "runAsEnabled": false,
        "runAsDefaultUser": "",
        "shellProfile": {
          "windows": "",
          "linux": "user=$(id -un 1000 2>/dev/null); [ -n \"$user\" ] && sudo su - $user || /bin/bash"
        }
      }
    }' \
    --region "$REGION"
fi

echo "✓ Configuration complete for $REGION"
