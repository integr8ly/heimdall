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

### Sample Output

```
---- CHECK RESULT ----
Checking image registry.redhat.io/amq7/amq-online-1-address-space-controller:1.2
Running hash:  sha256:034f245f47051215834f477713fd3c976d2f28fa3763d8958e984461d786985b
Latest Registry  hash:  sha256:034f245f47051215834f477713fd3c976d2f28fa3763d8958e984461d786985b err <nil>
** RUNNING IMAGE IS UPTO DATE **
-- END RESULT ---
---- CHECK RESULT ----
Checking image registry.redhat.io/amq7/amq-online-1-standard-controller:1.2
Running hash:  sha256:f481d9d02a30e9c81d001dabd11541fd361bd01cbcb3ddd03728618bad5a8dff
Latest Registry  hash:  sha256:f481d9d02a30e9c81d001dabd11541fd361bd01cbcb3ddd03728618bad5a8dff err <nil>
** RUNNING IMAGE IS UPTO DATE **
-- END RESULT ---
---- CHECK RESULT ----
Checking image registry.redhat.io/amq7/amq-online-1-agent:1.2
Running hash:  sha256:e490d3d687562f72629d0b11eddd9b287e1ed7139e860918cb73c84ee1f69314
Latest Registry  hash:  sha256:e490d3d687562f72629d0b11eddd9b287e1ed7139e860918cb73c84ee1f69314 err <nil>
** RUNNING IMAGE IS UPTO DATE **
-- END RESULT ---
---- CHECK RESULT ----
Checking image registry.redhat.io/amq7/amq-online-1-api-server:1.2
Running hash:  sha256:343f921fe5e328a526513bc8ab6a4f25e962a4f5441d8d445b38b49e9412ef72
Latest Registry  hash:  sha256:343f921fe5e328a526513bc8ab6a4f25e962a4f5441d8d445b38b49e9412ef72 err <nil>
** RUNNING IMAGE IS UPTO DATE **
-- END RESULT ---
---- CHECK RESULT ----
Checking image docker.io/openshift/oauth-proxy:latest
Running hash:  sha256:6bc1759a3202b4614739f12441461e344907f6b3f758c34314284debe36d4e15
Latest Registry  hash:  sha256:6bc1759a3202b4614739f12441461e344907f6b3f758c34314284debe36d4e15 err <nil>
** RUNNING IMAGE IS UPTO DATE **
-- END RESULT ---
---- CHECK RESULT ----
Checking image registry.redhat.io/amq7/amq-online-1-console-httpd:1.2
Running hash:  sha256:96d2c54eaf6e1bbcea11d09bd05ff194c11d0e880f87f5b8f9f529ee203be313
Latest Registry  hash:  sha256:96d2c54eaf6e1bbcea11d09bd05ff194c11d0e880f87f5b8f9f529ee203be313 err <nil>
** RUNNING IMAGE IS UPTO DATE **
-- END RESULT ---
---- CHECK RESULT ----
Checking image registry.redhat.io/amq7/amq-online-1-controller-manager:1.2
Running hash:  sha256:395d269ad4c8434f9b158e5dd3caa1bf27d64534bde6d9ba8682e69a14fab812
Latest Registry  hash:  sha256:395d269ad4c8434f9b158e5dd3caa1bf27d64534bde6d9ba8682e69a14fab812 err <nil>
** RUNNING IMAGE IS UPTO DATE **
-- END RESULT ---
---- CHECK RESULT ----
Checking image registry.redhat.io/redhat-sso-7/sso73-openshift:latest
Running hash:  sha256:35740d1dbebbb4dc39ea9ce4736d5cc54675a984b1ec0f9bef67eb48e93ffe2d
Latest Registry  hash:  sha256:35740d1dbebbb4dc39ea9ce4736d5cc54675a984b1ec0f9bef67eb48e93ffe2d err <nil>
** RUNNING IMAGE IS UPTO DATE **
-- END RESULT ---
---- CHECK RESULT ----
Checking image registry.redhat.io/amq7/amq-online-1-service-broker:1.2
Running hash:  sha256:4c4c63105220ec5be952175fd6f84576821aa97f993a458183c5539716bec1a8
Latest Registry  hash:  sha256:4c4c63105220ec5be952175fd6f84576821aa97f993a458183c5539716bec1a8 err <nil>
** RUNNING IMAGE IS UPTO DATE **
-- END RESULT ---
---- CHECK RESULT ----
Checking image registry.redhat.io/rhscl/postgresql-96-rhel7:latest
Running hash:  sha256:ffd9b8e71e72a351464f54c3ad2f9151c17a45c089aeb4db1083f9c1c3c3a142
Latest Registry  hash:  sha256:ffd9b8e71e72a351464f54c3ad2f9151c17a45c089aeb4db1083f9c1c3c3a142 err <nil>
** RUNNING IMAGE IS UPTO DATE **
-- END RESULT ---
```
