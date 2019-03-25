DD_API_KEY ?= ""


DD_AGENT_HOST ?= "127.0.0.1"
DD_AGENT_PORT ?= 8126
DD_SERVICE_NAME ?= "todo-app"
DD_SPAN_TAGS ?= "env: test"

run-dd-agent:
	@docker run -d \
	-v /var/run/docker.sock:/var/run/docker.sock:ro \
	-v /proc/:/host/proc/:ro \
	-v /sys/fs/cgroup/:/host/sys/fs/cgroup:ro \
	-e DD_API_KEY=$(DD_API_KEY) \
	-e DD_APM_ENABLED=true \
	-p 8126:8126 \
	datadog/agent:latest

run:
	go run ./cmd/todo-list-server/

.PHONY: vendor
vendor: ## Updates the vendoring directory.
	@$(RM) go.sum
	@$(RM) -r vendor
	GO111MODULE=on go mod init || true
	GO111MODULE=on go mod tidy
	GO111MODULE=on go mod vendor
	@$(RM) Gopkg.toml Gopkg.lock