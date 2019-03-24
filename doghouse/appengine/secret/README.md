## Encrypt files to deploy the server automatically

https://console.cloud.google.com/security/kms/keyring/manage/global/reviewdog-doghouse-deploy?project=review-dog&folder&organizationId

```shell
$ gcloud kms encrypt --location=global --keyring="reviewdog-doghouse-deploy" --key="secret-env" --ciphertext-file=encrypted-secret.yaml.bin --plaintext-file=secret.yaml
$ gcloud kms encrypt --location=global --keyring="reviewdog-doghouse-deploy" --key="secret-env" --ciphertext-file=encrypted-reviewdog.private-key.pem.bin --plaintext-file=reviewdog.private-key.pem
```
