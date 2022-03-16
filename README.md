# default-imagepullsecrets

The admission controller that applies default imagePullSecrets to pods.

## Getting Started

### Install cert-manager
default-imagepullsecrets uses [cert-manager](https://cert-manager.io/docs/) for
certificate management of Admission Webhook. Make sure you have already
installed cert-manager before you start.

- [Install cert-manager on kubernetes](https://cert-manager.io/docs/installation/)

### Deploy default-imagepullsecrets
```sh
export VERSION=v0.1.1
export DEFAULT_IMAGEPULLSECRETS="mysecret0,my-secret1"
envsubst < default-imagepullsecrets.yaml | kubectl apply -f -

# Wait for default-imagepullsecrets to be rollout
kubectl rollout status -n default-imagepullsecrets deployment default-imagepullsecrets

# Deploy MutatingWebHookConfiguration
kubectl apply -f mutating-webhook-configuration.yaml
```

```sh
# Run a test pod
kubectl run test --image=nginx
# Make sure that default imagePullSecrets are applied
kubectl get pods test -o jsonpath='{.spec.imagePullSecrets}{"\n"}'
```

### Clean up
```sh
kubectl delete -f mutating-webhook-configuration.yaml

export VERSION=v0.1.1
export DEFAULT_IMAGEPULLSECRETS="mysecret0,my-secret1"
envsubst < default-imagepullsecrets.yaml | kubectl delete -f -
```
