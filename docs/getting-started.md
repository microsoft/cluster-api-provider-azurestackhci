## Prerequisites

### Requirements

- Linux, macOS or Windows with WSL.
- Install the [Kubernetes CLI](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [KIND]
- [kustomize]
- make
- gettext (with `envsubst` in your PATH)

### Optional

- [Homebrew][brew] (MacOS)
- [jq]
- [Go]

[brew]: https://brew.sh/
[go]: https://golang.org/dl/
[jq]: https://stedolan.github.io/jq/download/
[kind]: https://sigs.k8s.io/kind
[kustomize]: https://github.com/kubernetes-sigs/kustomize

### Setting up your AzureStackHCI environment

There is an assumption that the AzureStackHCI environment is fully deployed and the MOC FQDN is reachable from your dev environment.

#### Provision MOC image gallery with kubernetes vhd

Download the Kubernetes VHD from <DOWNLOAD_LINK_TO_COME>. Lets say we have `Linux_k8s_1-18-6.vhdx`

```bash
echo "name: Linux_k8s_1-18-6" > .\galleryimage.yaml
mocctl compute galleryimage create --config  --image-path "\path\to\Linux_k8s_1-18.6.vhdx" --location <YOUR_HCI_LOCATION>
```

#### Provision ClusterAPI AzureStackHCI Identity

Then you need to create an identity so that ClusterAPI AzureStackHCI can talk to the MOCStack.

```bash
echo "name: caphuser" > .\caphuser.yaml
mocctl security identity create --config .\caphuser.yaml
mocctl security identity get --name zach --query [*].token > identity-token.yaml
```

Then using the name, token and certificate of the endpoint, you will need to create a `moclogintoken.yaml` which will look like...

```yaml
name: caphuser
token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9....
certificate: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t...
```

Then you will need to embed this into a kubernetes secret and place the secret in the caph-system namespace...

```yaml
apiVersion: v1
data:
  value: <BASE64 encoded moclogintoken.yaml>
kind: Secret
metadata:
  name: moclogintoken
  namespace: caph-system
type: Opaque
```

## Getting Started with KIND

Install clusterctl binary.

```
curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v0.3.9/clusterctl-linux-amd64 -o clusterctl
chmod +x ./clusterctl
sudo mv ./clusterctl /usr/local/bin/clusterctl
clusterctl version
```

Set the following environment variables.
```
export AZURESTACKHCI_CLOUDAGENT_FQDN=<CLOUDAGENT_FQDN>
export AZURESTACKHCI_BINARY_LOCATION=<AZURESTACKHCI_BINARY_LOCATION>
export AZURESTACKHCI_CLUSTER_RESOURCE_GROUP=<GROUP>
export AZURESTACKHCI_VNET_NAME=External
export KUBERNETES_VERSION=v1.18.6
export WORKER_MACHINE_COUNT=1
```

Finally run

```
make create-cluster
```

This will create a KIND Management Cluster and also create a target cluster.
