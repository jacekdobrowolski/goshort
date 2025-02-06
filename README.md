## Link shortener application
Simple link shortener application written in go with opentelmetry instrumentation
![example trace screenshot](https://private-user-images.githubusercontent.com/44121786/410589660-a958ca13-6f83-47ce-a8ea-3dac4404ac81.png?jwt=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbSIsImtleSI6ImtleTUiLCJleHAiOjE3Mzg4NjY1NDksIm5iZiI6MTczODg2NjI0OSwicGF0aCI6Ii80NDEyMTc4Ni80MTA1ODk2NjAtYTk1OGNhMTMtNmY4My00N2NlLWE4ZWEtM2RhYzQ0MDRhYzgxLnBuZz9YLUFtei1BbGdvcml0aG09QVdTNC1ITUFDLVNIQTI1NiZYLUFtei1DcmVkZW50aWFsPUFLSUFWQ09EWUxTQTUzUFFLNFpBJTJGMjAyNTAyMDYlMkZ1cy1lYXN0LTElMkZzMyUyRmF3czRfcmVxdWVzdCZYLUFtei1EYXRlPTIwMjUwMjA2VDE4MjQwOVomWC1BbXotRXhwaXJlcz0zMDAmWC1BbXotU2lnbmF0dXJlPWVkMzdhYTJkZjMwMGUzNWFlMzJkZTQ2YWI5MDUzM2RiMjIwZGU5MzhlMGNiZTk2ZjkwYjFmOTU4ZmU3MjA4NDMmWC1BbXotU2lnbmVkSGVhZGVycz1ob3N0In0.S16tjn557Ip3VJf3fqKpdobIUuug2kst4IyK7zxHKpo)

# Useful commands

## Setup

Working Kind and skaffold are required.
Networking is for now solved using cloud-provider-kind which is separate binary that has to run on host along side kind cluster
```bash
go install sigs.k8s.io/cloud-provider-kind@latest

kind-cloud-provider
```

Kind cluster setup
```bash
kind cluster create --config=kind-config.yaml
kubectl create secret generic postgres --from-literal=password=replace-with-your-password
skaffold run
```
or 'make run' 'make clean' for cleanup

## Tests

simple k6 test
```bash
docker run --network=host -e LINKS_HOST=$(kubectl get svc links-service -o=jsonpath='{.status.loadBalancer.ingress[*].ip}') --rm -v ./tests/add_and_get:/scripts grafana/k6 run /scripts/test.js

```
