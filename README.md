# pingdom-operator

Kubernetes Operator that automates creating Pingdom checks for Ingress resources.

## Usage

The Ingress resource needs to include the annotation monitoring.rossfairbanks.com/pingdom
A Pingdom HTTP Check will be created for each hostname in the Ingress.

e.g.

```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: pets
  annotations:
    monitoring.rossfairbanks.com/pingdom: "pets"
```

## Installation

* Register with Pingdom and create an API key.
* Edit pingdom-secret.yaml and set the API user, password and key.
* Create the secret.

```
$ kubectl apply -f pingdom-secret.yaml
```

* Create the operator.

```
$ kubectl apply -f deployment.yaml
```

Operator will also create a Third Party Resource upon start.

## Building

Build the Go binary and Docker image. Developed using Go 1.7 and Kubernetes
1.5 using Minikube.

```
$ make
```

## Run tests

Run the unit tests.

```
$ make test
```
