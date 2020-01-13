## What it does

- Figure out the images in use for a particular namespaces or an individual deployment/deploymentconfig whether those images are part of image streams, image stream tags or direct image references. 
- It will then check the redhat registry for that image and get the image digest from the registry and check it against the digest for the image running in the cluster.
- If it is out of date, it will figure out which none floating tag is being used in the cluster and use the registry API to figure out which CVEs are fixed by the newer image
- It will then label pods with this information so alerting can happen based on these labels


## Try it

- Install Go (go1.13)
- Clone this repo

```
cd <clone_dir>/cmd/cli
go build .

```

Login to a target cluster to ensure your kube config is pointing to the correct cluster, the tool uses the local kube config

You need to get a service account token and login with it locally using the docker login method. Here is one I have set up (you need to login to see it) https://access.redhat.com/terms-based-registry/#/token/-heimdall

``` 
./cli -namespaces=fuse
```

### Sample Output

```
+---------------------+-----------------------------------------------------+--------------+---------+--------------------+----------------------+----------------------+--------------+--------------------+-----------------------------+---------------+----------------+---------------+
| COMPONENT           | IMAGE                                               | IMAGE STREAM | TAG     | UPTO DATE WITH TAG | PERSISTENT IMAGE TAG | LATEST PATCH TAG     | FLOATING TAG | USING FLOATING TAG | UPTO DATE WITH FLOATING TAG | CRITICAL CVES | IMPORTANT CVES | MODERATE CVES |
+---------------------+-----------------------------------------------------+--------------+---------+--------------------+----------------------+----------------------+--------------+--------------------+-----------------------------+---------------+----------------+---------------+
| broker-amq          | jboss-amq-6/amq63-openshift                         | true         | 1.3     | true               | 1.3-7                | 1.3-7                | 1.3          | true               | true                        |             0 |              0 |             0 |
| komodo-server       | fuse7-tech-preview/data-virtualization-server-rhel7 | true         | 1.4-15  | true               | 1.4-15               | 1.4-15.1567588155    | 1.4          | false              | false                       |             0 |              1 |            40 |
| syndesis-db         | rhscl/postgresql-95-rhel7                           | true         | latest  | true               | 9.5-44               | 9.5-44               | latest       | true               | true                        |             0 |              0 |             0 |
| syndesis-db         | fuse7-tech-preview/fuse-postgres-exporter           | true         | 1.4-4   | true               | 1.4-4                | 1.4-4                | 1.4          | false              | true                        |             0 |              0 |             0 |
| syndesis-meta       | fuse7/fuse-ignite-meta                              | true         | 1.4-13  | true               | 1.4-13               | 1.4-16               | 1.4          | false              | false                       |             0 |             15 |            55 |
| syndesis-oauthproxy | openshift4/ose-oauth-proxy                          | true         | 4.1     | true               | v4.1.22-201910291109 | v4.1.22-201910291109 | 4.1          | true               | true                        |             0 |              0 |             0 |
| syndesis-operator   | fuse7/fuse-online-operator                          | true         | 1.4-16  | true               | 1.4-16               | 1.4-16               | 1.4          | false              | true                        |             0 |              0 |             0 |
| syndesis-prometheus | openshift3/prometheus                               | true         | v3.9.25 | true               | v3.9.25              | v3.9.102-1           | v3.9.25      | true               | true                        |             0 |             18 |            43 |
| syndesis-server     | fuse7/fuse-ignite-server                            | true         | 1.4-17  | true               | 1.4-17               | 1.4-17               | 1.4          | false              | true                        |             0 |              0 |             0 |
| syndesis-ui         | fuse7/fuse-ignite-ui                                | true         | 1.4-9   | true               | 1.4-9                | 1.4-9                | 1.4          | false              | true                        |             0 |              0 |             0 |
+---------------------+-----------------------------------------------------+--------------+---------+--------------------+----------------------+----------------------+--------------+--------------------+-----------------------------+---------------+----------------+---------------+

```

RANDOM UPDATE DO NOT MERGE
