#!/bin/bash

set -e

NAMESPACE="observability-copilot"
DEPLOYMENT="backend"
IMAGE_NAME="your-backend-image:latest"

echo "📌 Scaling down deployment..."
kubectl scale deploy "$DEPLOYMENT" -n "$NAMESPACE" --replicas=0

echo "📌 Removing old image from Minikube..."
minikube image rm "$IMAGE_NAME" || true

echo "📌 Building Docker image..."
docker build -t "$IMAGE_NAME" .

echo "📌 Loading image into Minikube..."
minikube image load "$IMAGE_NAME"

echo "📌 Scaling deployment back up..."
kubectl scale deploy "$DEPLOYMENT" -n "$NAMESPACE" --replicas=1

echo "🎉 Done! Backend rebuilt and redeployed."

