clean:
	kind delete cluster --name goshort

create-secret: create-cluster
	kubectl create secret generic postgres --from-literal=password=$$(head -c 16 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9')

create-cluster:
	kind create cluster --name goshort --config kind-config.yaml

run: create-cluster create-secret
	skaffold run
