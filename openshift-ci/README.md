## OpenShift CI

### Dockerfile.tools

Base image used on CI for all builds and test jobs.

#### Build and Test

```
$ docker build -t registry.svc.ci.openshift.org/openshift/release:intly-golang-1.12 - < Dockerfile.tools
$ IMAGE_NAME=registry.svc.ci.openshift.org/openshift/release:intly-golang-1.12 test/run
operator-sdk version: v0.10.0, commit: ff80b17737a6a0aade663e4827e8af3ab5a21170
go version go1.12.9 linux/amd64
hello world
SUCCESS!
```