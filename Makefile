.DEFAULT_GOAL := help
.EXPORT_ALL_VARIABLES:

debug: ## build precompiled server for debug
	go build -gcflags=all="-N -l" -o open-restconfd main.go response.go route.go error.go utilities.go

build: ## build restconf server
	go build -o open-restconfd main.go response.go route.go error.go utilities.go

run: build ## run restconf server
	./open-restconfd -f modules/example/example-jukebox.yang -f modules/example/example-ops.yang \
	--startup-format yaml --startup testdata/jukebox.yaml

watch: ## hot-reloading
	reflex -s -r '\.go$$' make run

help: ## print command help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'