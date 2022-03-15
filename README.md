# default-imagepullsecrets

The admission controller that applies default imagePullSecrets to pods.

## Getting Started

By default, default-imagepullsecrets uses
[cert-manager](https://cert-manager.io/docs/) for certificate management of
Admission Webhook. Make sure you have already installed cert-manager before you
install.

- [Install cert-manager on kubernetes](https://cert-manager.io/docs/installation/)

**Deploy default-imagepullsecrets**
```sh
export DEFAULT_IMAGEPULLSECRETS="mysecret0,my-secret1"
envsubst < default-imagepullsecrets.yaml | kubectl apply -f -
```
