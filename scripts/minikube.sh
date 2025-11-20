#!/usr/bin/env bash

set -ex

minikube delete
minikube start
eval $(minikube docker-env)
make docker-build IMG=bindplane-operator:local
make install
make deploy