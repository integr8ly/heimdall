## OpenShift CI

### Dockerfile.tools

Base image used on CI for all builds and test jobs. See [here](https://github.com/integr8ly/ci-cd/blob/master/openshift-ci/README.md) for more information on creating and deploying a new image.

#### Build and Test

```
$ docker build -t registry.svc.ci.openshift.org/integr8ly/heimdall-base-image:latest - < Dockerfile.tools
$ IMAGE_NAME=registry.svc.ci.openshift.org/integr8ly/heimdall-base-image:latest test/run
operator-sdk version: "v0.12.0", commit: "2445fcda834ca4b7cf0d6c38fba6317fb219b469", go version: "go1.13.5 linux/amd64"                                                                                            
go version go1.13.5 linux/amd64  
...
SUCCESS!
```