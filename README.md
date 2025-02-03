## Link shortener application

```bash
go install sigs.k8s.io/cloud-provider-kind@latest
kind cluster create --config=kind-config.yaml
kind-cloud-provider
kubectl create secret generic postgres --from-literal=password=replace-with-your-password
skaffold run
```
