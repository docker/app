KUBECONFIG=/root/.kube/config

CNAB_ACTION ?= status
CNAB_INSTALLATION_NAME ?= helmtest

CNAB_P_HELM_OPTIONS ?= --dry-run --debug
CNAB_P_NAMESPACE ?= hellohelm

CHART=/cnab/app/charts/alpine

install:
	@echo "Do $(CNAB_ACTION) for $(CNAB_INSTALLATION_NAME)"
	helm install --namespace $(CNAB_P_NAMESPACE) -n $(CNAB_INSTALLATION_NAME) $(CHART)

uninstall: 
	@echo "Do Uninstall"
	helm delete --purge $(CNAB_INSTALLATION_NAME)

upgrade:
	@echo "Do Upgrade"
	helm upgrade --namespace $(CNAB_P_NAMESPACE) $(CNAB_INSTALLATION_NAME) $(CHART)

status:
	@echo "Do Status"
	helm status $(CNAB_INSTALLATION_NAME)

.PHONY: install uninstall upgrade status
