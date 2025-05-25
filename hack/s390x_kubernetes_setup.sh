#!/bin/bash

# Ensure script runs as root
if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root!" >&2
    exit 1
fi

sleep 30
export DEBIAN_FRONTEND=noninteractive

# Update and upgrade system
apt-get update -y
apt-get upgrade -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"

# Add kernel modules required for Kubernetes
tee /etc/modules-load.d/containerd.conf <<EOF
overlay
br_netfilter
EOF

modprobe overlay
modprobe br_netfilter

# Set up sysctl params for Kubernetes networking
tee /etc/sysctl.d/kubernetes.conf <<EOF
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward = 1
EOF

sysctl --system

# Disable swap (required for Kubernetes)
swapoff -a
sed -i '/ swap / s/^/#/' /etc/fstab

# Install required dependencies
apt install -y curl gnupg2 software-properties-common apt-transport-https ca-certificates

# Install containerd
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmour -o /etc/apt/trusted.gpg.d/docker.gpg
chmod 644 /etc/apt/trusted.gpg.d/docker.gpg
add-apt-repository "deb [arch=s390x] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

apt update
apt install -y containerd.io

# Configure containerd
containerd config default | tee /etc/containerd/config.toml >/dev/null 2>&1
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/g' /etc/containerd/config.toml

systemctl restart containerd
systemctl enable containerd

# Install Kubernetes components
mkdir -p /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.30/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
chmod 644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.30/deb/ /' | tee /etc/apt/sources.list.d/kubernetes.list
chmod 644 /etc/apt/sources.list.d/kubernetes.list

apt-get update
apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl

# Initialize Kubernetes cluster
IPADDR=$(curl -s ifconfig.me)
NODENAME=$(hostname -s)
POD_CIDR="10.244.0.0/16"

kubeadm init --control-plane-endpoint=$IPADDR --apiserver-cert-extra-sans=$IPADDR --pod-network-cidr=$POD_CIDR --node-name $NODENAME --ignore-preflight-errors Swap

sleep 30

# Configure kubectl for root user
mkdir -p $HOME/.kube
cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
chown -R $(id -u):$(id -g) $HOME/.kube

# Install Flannel CNI for networking
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Remove master taint to allow workloads to run on this single-node cluster
kubectl taint nodes $(hostname) node-role.kubernetes.io/control-plane:NoSchedule-


# Display cluster status
kubectl get nodes
kubectl get pods -A
