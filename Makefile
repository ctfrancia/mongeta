.PHONY: run

run:
	export MONGETA_WORKER_HOST=localhost && \
	export MONGETA_WORKER_PORT=5556 && \
	export MONGETA_HOST=localhost && \
	export MONGETA_PORT=5555 && \
	go run main.go
