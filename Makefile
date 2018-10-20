TAG?=latest
.PHONY: build

build:
	docker build -t affixxx/sidekiq-connector:$(TAG) .
push:
	docker push affixxx/sidekiq-connector:$(TAG)
