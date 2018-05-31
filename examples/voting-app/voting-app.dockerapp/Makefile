# Input.
SETTINGS_DIR ?= settings
APP_NAME := voting-app

# Output.
DEVELOPMENT_DIR := build/development
PRODUCTION_DIR := build/production
PACK := $(APP_NAME).pack

#
# Cleanup.
#
cleanup/production:
	@rm -rf $(PRODUCTION_DIR)

cleanup/development:
	@rm -rf $(DEVELOPMENT_DIR)

cleanup: cleanup/production cleanup/development

#
# Render.
#
render/production: cleanup/production
	@mkdir -p $(PRODUCTION_DIR)
	docker-app render --settings-files $(SETTINGS_DIR)/production.yml > $(PRODUCTION_DIR)/docker-compose.yml

render/development: cleanup/development
	@mkdir -p $(DEVELOPMENT_DIR)
	docker-app render- --settings-files $(SETTINGS_DIR)/development.yml > $(DEVELOPMENT_DIR)/docker-compose.yml

render: render/production render/development

#
# Stop.
#
stop/production:
	docker stack rm ${APP_NAME}

stop/development:
	docker stack rm ${APP_NAME}-dev

stop: stop/production stop/development

#
# Deploy.
#
deploy/production: render/production stop/production
	docker-app deploy --settings-files $(SETTINGS_DIR)/production.yml

deploy/development: render/development stop/development
	docker-app deploy --settings-files $(SETTINGS_DIR)/development.yml

#
# Pack.
#
pack:
	docker-app pack -o $(PACK)

#
# Helm.
#
helm/production:
	docker-app helm --settings-files $(SETTINGS_DIR)/production.yml

helm/development:
	docker-app helm --settings-files $(SETTINGS_DIR)/development.yml
