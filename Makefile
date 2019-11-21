ORG=integreatly
PROJECT=heimdall-operator
REG=quay.io
TAG=master
COMPILE_TARGET=./tmp/_output/bin/$(PROJECT)
NAMESPACE=""

SHELL=/bin/bash

.PHONY: code/gen
code/gen:
	operator-sdk generate k8s
	@go generate ./...

.PHONY: code/check
code/check:
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)

.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: setup/moq
setup/moq:
	dep ensure
	cd vendor/github.com/matryer/moq/ && go install .

.PHONY: code/run
code/run:
	@operator-sdk up local --namespace=$(NAMESPACE)

.PHONY: test/unit
test/unit:
	@./scripts/ci/unit_test.sh

.PHONY: image/build
image/build:
	@operator-sdk build $(REG)/$(ORG)/$(PROJECT):$(TAG)

.PHONY: image/push
image/push:
	docker push $(REG)/$(ORG)/$(PROJECT):$(TAG)

.PHONY: image/build/push
image/build/push: image/build image/push

.PHONY: cluster/prepare/local
cluster/prepare/local:
	-oc create -f deploy/crds/*_crd.yaml
	@oc create -f deploy/service_account.yaml
	@oc create -f deploy/role.yaml
	@oc create -f deploy/role_binding.yaml
