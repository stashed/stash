#!/bin/bash
set -x

kubectl delete deployment -l app=stash -n kube-system
kubectl delete service -l app=stash -n kube-system
kubectl delete secret -l app=stash -n kube-system

# Delete RBAC objects, if --rbac flag was used.
kubectl delete serviceaccount -l app=stash -n kube-system
kubectl delete clusterrolebindings -l app=stash -n kube-system
kubectl delete clusterrole -l app=stash -n kube-system

kubectl delete initializerconfiguration -l app=stash
