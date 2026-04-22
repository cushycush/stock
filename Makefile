VERSION ?= v0.3.1-dev

# Container runtime: prefer podman (common on Arch), fall back to docker.
CONTAINER ?= $(shell command -v podman >/dev/null 2>&1 && echo podman || echo docker)

# Target distro for `make dogfood`. Override with `make dogfood DISTRO=fedora`.
DISTRO ?= ubuntu
IMAGE_ubuntu := ubuntu:24.04
IMAGE_debian := debian:12
IMAGE_fedora := fedora:41
IMAGE_alpine := alpine:3.20
IMAGE_arch   := archlinux:latest
IMAGE := $(or $(IMAGE_$(DISTRO)),$(IMAGE_ubuntu))

build:
	go build -ldflags "-X main.version=$(VERSION)" -o stock ./cmd/stock

# Spin up a throwaway container with the freshly-built binary + a fixture
# dotfiles repo mounted, and drop into an interactive shell. Run stock
# against real package managers in a clean-slate env so you find the
# shell-out bugs unit tests miss.
#
#   make dogfood                  # ubuntu by default
#   make dogfood DISTRO=fedora    # other distros
#   make dogfood CONTAINER=docker # force docker over podman
#
# Inside the shell try: `stock platform`, `stock doctor`, `stock diff`,
# `stock install`, `stock bootstrap`. Exit when done — the container is
# destroyed with --rm.
dogfood: build
	@command -v $(CONTAINER) >/dev/null 2>&1 || { \
		echo "error: '$(CONTAINER)' not found on \$$PATH"; \
		echo; \
		echo "install a container runtime:"; \
		echo "  Arch:    sudo pacman -S podman"; \
		echo "  Debian:  sudo apt install podman   (or docker.io)"; \
		echo "  Fedora:  sudo dnf install podman"; \
		echo "  macOS:   brew install podman && podman machine init && podman machine start"; \
		echo; \
		echo "already have one under a different name? override: make dogfood CONTAINER=docker"; \
		exit 1; \
	}
	@test -x ./stock || { echo "build failed"; exit 1; }
	@echo "==> dogfooding stock on $(DISTRO) ($(IMAGE)) via $(CONTAINER)"
	$(CONTAINER) run --rm -it \
		-v $(CURDIR)/stock:/usr/local/bin/stock:Z \
		-v $(CURDIR)/hack/dogfood:/root/dotfiles:Z \
		-w /root/dotfiles \
		-e FORCE_COLOR=1 \
		$(IMAGE) bash

clean:
	rm -f stock

.PHONY: build dogfood clean
