# hostroutes for Kubernetes

hostroutes is a super light network layer for pod network. hostroutes runs on every node and updates their local route table as needed.

## Why hostroutes

hostroutes works similarly as flannel hostgw mode does without etcd.
It only connects to the k8s api server and authenticates the same way the kubelet does, which is more secure and efficient. And hostroutes doesn't allocate subnets. It merely routes them.

## Requirements

Kubernetes 1.3+

## How

1. **Assign a pod cidr to every node.** You can do this either by passing `--allocate-node-cidrs=true --configure-cloud-routes=false` to the k8s controller manager, or by doing your own subnet allocating and putting it in node.PodCidr.

2. **Run hostroutes on every node.** If you don't feel like setting up it manually you can use a daemonset.

