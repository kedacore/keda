#!/bin/bash

# Ensure script runs as root
if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root!" >&2
    exit 1
fi

echo "Draining and deleting nodes..."
kubectl drain $(hostname) --ignore-daemonsets --delete-emptydir-data
kubectl delete node $(hostname)

echo "Resetting Kubernetes..."
kubeadm reset -f

echo "Removing Kubernetes packages..."
apt-get purge -y kubelet kubeadm kubectl
apt-get autoremove -y
rm -rf /etc/kubernetes
rm -rf ~/.kube

echo "Stopping and disabling containerd..."
systemctl stop containerd
systemctl disable containerd
apt-get purge -y containerd.io
rm -rf /etc/containerd

echo "Removing Kubernetes repositories..."
rm -f /etc/apt/sources.list.d/kubernetes.list
rm -f /etc/apt/keyrings/kubernetes-apt-keyring.gpg
rm -f /etc/apt/trusted.gpg.d/docker.gpg

echo "Flushing iptables rules..."
iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X

echo "Removing kernel module configurations..."
rm -f /etc/modules-load.d/containerd.conf
rm -f /etc/sysctl.d/kubernetes.conf
modprobe -r overlay
modprobe -r br_netfilter

echo "Re-enabling swap..."
sed -i '/ swap / s/^#//' /etc/fstab
swapon -a

echo "Cleaning up package lists..."
apt-get update -y

echo "Kubernetes cluster destroyed successfully."
