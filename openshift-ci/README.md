## OpenShift CI

### Dockerfile.tools

Base image used on CI for all builds and test jobs.

#### Build and Test

```
$ docker build -t registry.svc.ci.openshift.org/openshift/release:intly-golang-1.13 - < Dockerfile.tools
$ IMAGE_NAME=registry.svc.ci.openshift.org/openshift/release:intly-golang-1.13 test/run
operator-sdk version: "v0.12.0", commit: "2445fcda834ca4b7cf0d6c38fba6317fb219b469", go version: "go1.13.5 linux/amd64"
go version go1.13.5 linux/amd64
hello world
SUCCESS!
```