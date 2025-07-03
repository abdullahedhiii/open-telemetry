kubectl delete jobs --all --force | grep kibana
kubectl delete pods --all --force | grep kibana
kubectl delete secret kibana-service-es-token 
kubectl delete roles pre-install-kibana-service 
kubectl delete rolebindings pre-install-kibana-service
kubectl delete configmaps kibana-service-helm-scripts
kubectl delete serviceaccounts pre-install-kibana-service
kubectl delete service --all --force | grep kibana
