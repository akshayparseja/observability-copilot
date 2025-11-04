# 1. Add Helm repo
helm repo add nginx-stable https://helm.nginx.com/stable
helm repo update

# 2. Create namespace
kubectl create namespace ingress-nginx

# 3. Install NGINX Ingress Controller
helm install nginx-controller nginx-stable/nginx-ingress \
  --namespace ingress-nginx \
  --set controller.service.type=NodePort \
  --set controller.service.ports.http=30080 \
  --wait

# 4. Verify
kubectl get pods -n ingress-nginx
kubectl get svc -n ingress-nginx
