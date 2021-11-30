.DEFAULT_GOAL := help
.EXPORT_ALL_VARIABLES:

build: ## build restconf server
	go build -o server main.go

run: build ## run restconf server
	./server

watch: ## hot-reloading
	reflex -s -r '\.go$$' make run

help: ## print command help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'