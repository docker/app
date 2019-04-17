KUBECONFIG=/root/.kube/config

CNAB_ACTION ?= status
CNAB_INSTALLATION_NAME ?= helmtest

HELM_OPTIONS ?= --dry-run --debug
NAMESPACE ?= hellohelm

CHART=/cnab/app/charts/alpine

install:
	@echo "Do $(CNAB_ACTION) for $(CNAB_INSTALLATION_NAME)"
	helm install --namespace $(NAMESPACE) -n $(CNAB_INSTALLATION_NAME) $(CHART)

uninstall: 
	@echo "Do Uninstall"
	helm delete --purge $(CNAB_INSTALLATION_NAME)

upgrade:
	@echo "Do Upgrade"
	helm upgrade --namespace $(NAMESPACE) $(CNAB_INSTALLATION_NAME) $(CHART)

status:
	@echo "Do Status"
	helm status $(CNAB_INSTALLATION_NAME)

.PHONY: install uninstall upgrade status
