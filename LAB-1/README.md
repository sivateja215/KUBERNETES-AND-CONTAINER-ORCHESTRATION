# Experiment 1: Kubernetes Cluster and Application Management Using kind

## Aim

To create a multi-node Kubernetes cluster using kind, identify the control-plane components, deploy an application, scale it, expose it, and observe Kubernetes self-healing.

## Prerequisites

- Docker Desktop (Docker Engine running)
- kind
- kubectl
- Windows PowerShell
- Laptop with 8 GB RAM or more

## Folder Structure

```text
LAB-1/
│
├── README.md
├── kind-3node.yaml
|
└── httpd-deployment.yaml
```

## In-Lab Experiment

### Task 1: Create a 3-Node Kubernetes Cluster

The cluster contains:
- 1 Control Plane Node
- 2 Worker Nodes

**Commands:**

```powershell
kind create cluster --config kind-3node.yaml
kubectl get nodes
docker ps
kubectl get pods -n kube-system
```

**Expected Result:**

A three-node Kubernetes cluster is created successfully. The control-plane components can be inspected using kubectl.

### Task 2: Deploy and Manage NGINX

Create an NGINX Deployment with three replicas.

```powershell
kubectl create deployment web --image=nginx --replicas=3
kubectl get pods -o wide
```

**Self-Healing:**

```powershell
kubectl get pods -w
kubectl delete pod <pod-name>
```

Kubernetes automatically creates a replacement Pod to maintain the desired replica count.

**Scale the Deployment:**

```powershell
kubectl scale deployment web --replicas=5
kubectl get pods

kubectl scale deployment web --replicas=2
kubectl get pods
```

**Expose the Application:**

```powershell
kubectl expose deployment web --port=80
kubectl get services
kubectl port-forward service/web 8080:80
```

Open the application in your browser:

http://localhost:8080

## Post-Lab Experiment

### Task 1: Create a 4-Node Cluster

The cluster contains:
- 1 Control Plane Node
- 3 Worker Nodes

Delete the previous cluster before creating the new one.

```powershell
kind delete cluster --name klu
kind create cluster --config kind-4node.yaml
kubectl get nodes
```

### Task 2: Deploy HTTPD with Four Replicas

Apply the HTTPD Deployment YAML file.

```powershell
kubectl apply -f httpd-deployment.yaml
kubectl get pods -o wide
```

Observe the four HTTPD Pods and their node placement.

### Task 3: Worker Node Failure

Stop one worker node:

```powershell
docker stop klu-worker2
```

Observe node and Pod status:

```powershell
kubectl get nodes
kubectl get pods -o wide
```

Observe the changes for approximately six minutes.

Restart the worker:

```powershell
docker start klu-worker2
```

Verify recovery:

```powershell
kubectl get nodes
kubectl get pods -o wide
```

### Task 4: Compare Docker and Kubernetes

**Run NGINX using Docker:**

```powershell
docker run -d --name my-nginx -p 8081:80 nginx
```

**Run NGINX using Kubernetes:**

```powershell
kubectl create deployment my-nginx --image=nginx --replicas=3
kubectl get pods
```

**Three Kubernetes capabilities:**

1. Self-healing – replaces failed Pods to maintain the desired number of replicas.
2. Scaling – increases or decreases the number of application replicas.
3. Service discovery and load balancing – provides a stable way to access application Pods.

## YAML Files Description

| File | Purpose |
|---|---|
| `kind-3node.yaml` | Creates a cluster with one control plane and two workers. |
| `kind-4node.yaml` | Creates a cluster with one control plane and three workers. |
| `httpd-deployment.yaml` | Deploys HTTPD with four replicas. |

## Result

Successfully created multi-node Kubernetes clusters using kind, identified control-plane Pods, deployed and scaled NGINX, tested self-healing, exposed the application, deployed HTTPD, and observed worker node failure and recovery.

## Conclusion

This experiment demonstrates Kubernetes cluster creation, application deployment, scaling, service exposure, self-healing, and worker node management using kind, kubectl, and Docker.