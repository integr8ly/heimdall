ORG=integreatly
PROJECT=heimdall-operator
REG=quay.io
TAG=master
COMPILE_TARGET=./tmp/_output/bin/$(PROJECT)
NAMESPACE=heimdall

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

.PHONY: code/compile
code/compile:
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o=$(COMPILE_TARGET) ./cmd/manager

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

.PHONY: test/e2e
test/e2e:
	@echo Running e2e tests: TODO

.PHONY: image/build
image/build:
	@operator-sdk build $(REG)/$(ORG)/$(PROJECT):$(TAG)

.PHONY: image/push
image/push:
	docker push $(REG)/$(ORG)/$(PROJECT):$(TAG)

.PHONY: image/build/push
image/build/push: image/build image/push

.PHONY: cluster/prepare
cluster/prepare:
	-oc create namespace $(NAMESPACE)
	-oc create -f deploy/crds/*_crd.yaml -n $(NAMESPACE)
	@oc create -f deploy/service_account.yaml -n $(NAMESPACE)
	@oc create -f deploy/role.yaml -n $(NAMESPACE)
	@oc create -f deploy/role_binding.yaml -n $(NAMESPACE)

.PHONY: cluster/clean
cluster/clean:
	-oc delete namespace $(NAMESPACE)
	-oc delete -f deploy/crds/*_crd.yaml -n $(NAMESPACE)
