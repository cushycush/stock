VERSION ?= v0.3.1-dev

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
#   make dogfood DISTRO=fedora    # other distros: ubuntu · debian · fedora · alpine · arch
#
# Inside the shell try: `stock platform`, `stock doctor`, `stock diff`,
# `stock install`, `stock bootstrap`. Exit when done — the container is
# destroyed with --rm.
dogfood: build
	@command -v docker >/dev/null 2>&1 || { \
		echo "error: docker not found on \$$PATH"; \
		echo; \
		echo "install it:"; \
		echo "  Arch:    sudo pacman -S docker && sudo systemctl enable --now docker"; \
		echo "           sudo usermod -aG docker \$$USER   # then log out + back in"; \
		echo "  Debian:  sudo apt install docker.io"; \
		echo "  Fedora:  sudo dnf install docker-ce"; \
		echo "  macOS:   brew install --cask docker"; \
		exit 1; \
	}
	@test -x ./stock || { echo "build failed"; exit 1; }
	@echo "==> dogfooding stock on $(DISTRO) ($(IMAGE))"
	docker run --rm -it \
		-v $(CURDIR)/stock:/usr/local/bin/stock \
		-v $(CURDIR)/hack/dogfood:/root/dotfiles \
		-w /root/dotfiles \
		-e FORCE_COLOR=1 \
		$(IMAGE) bash

clean:
	rm -f stock

.PHONY: build dogfood clean
