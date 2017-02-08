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

## Building

Build the Go binary and Docker image. Developed using Go 1.7 and Kubernetes
1.5 using Minikube.

```
$ make
```

## Enhancements - Pingdom third party resource

In a later version a Pingdom third party resource could be used
to configure the Pingdom checks.

e.g.

```
apiVersion: "pingdom.monitoring.rossfairbanks.com/v1alpha1"
kind: Pingdom
metadata:
  name: pets
spec:
  checkIntervalMins: 2
```
