# Shamelessly adapted from the Makefile at vieux/docker-volume-sshfs

PLUGIN_NAME=de-logging
PLUGIN_TAG=latest

all: clean docker rootfs create

clean:
	@echo "### Removing the ./plugin directory"
	@rm -rf ./plugin

docker:
	@echo "### Create the rootfs image"
	@docker build -t ${PLUGIN_NAME}:rootfs .

rootfs:
	@echo "### Create rootfs directory in ./plugin/rootfs"
	@mkdir -p ./plugin/rootfs
	@docker create --name rootfs-tmp ${PLUGIN_NAME}:rootfs
	@docker export rootfs-tmp | tar -x -C ./plugin/rootfs
	@echo "### Copy config.json to ./plugin/"
	@cp config.json ./plugin/
	@docker rm -fv rootfs-tmp

create:
	@echo "### Remove the ${PLUGIN_NAME} plugin from Docker"
	@docker plugin rm -f ${PLUGIN_NAME}:${PLUGIN_TAG} || true
	@echo "### Create the ${PLUGIN_NAME}:${PLUGIN_TAG} plugin from the contents of ./plugin/"
	@docker plugin create ${PLUGIN_NAME}:${PLUGIN_TAG} ./plugin

enable:
	@echo "### Enabling the ${PLUGIN_NAME}:${PLUGIN_TAG} plugin"
	@mkdir -p /var/log/de-docker-logging-plugin/
	@docker plugin enable ${PLUGIN_NAME}:${PLUGIN_TAG}

push: clean docker rootfs create enable
	@echo "### Push the ${PLUGIN_NAME}:${PLUGIN_TAG} plugin to the repository"
	@docker plugin push ${PLUGIN_NAME}:${PLUGIN_TAG}
