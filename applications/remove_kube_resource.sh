
kubectl delete roles pre-install-web-services-kibana post-delete-web-services-kibana
kubectl delete rolebindings pre-install-web-services-kibana post-delete-web-services-kibana
kubectl delete serviceaccounts pre-install-web-services-kibana post-delete-web-services-kibana 
kubectl delete jobs post-delete-web-services-kibana pre-install-web-services-kibana
kubectl delete pod pre-install-web-services-kibana-kvkqf  post-delete-web-services-kibana-bmrjp
