.PHONY: build_prod

build_prod:
	mage -v -ldflags "-X internal.Environment=prod"