# Very Alpha PoC for checking image freshness on a cluster


## What it does

It will figure out the images in use for a particular namespaces or an individual deployment/deploymentconfig, it takes
into acccount image streams and image stream tags. Once it has figured out the image and tag, it will then check the registry
the image came from and get the digest for that tag and check it against the digest for the image running in the cluster.
This tells you if the image running the cluster is up to date with the image at the same tag in the registry.


## Try it

- Install Go 
- Clone this repo

```
cd <clone_dir>/cmd/cli
go build .

```

Login to a target cluster to ensure your kube config is pointing to the correct cluster, the tool uses the local kube config

``` 
./cli check <namespace> 
```