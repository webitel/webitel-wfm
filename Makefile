# The real-Postgres tests (infra/storage/dbsql) skip unless WFM_TEST_POSTGRES_DSN is set.
# test-pg starts a throwaway Postgres in docker, runs the whole suite against it, and removes it.

PG_IMAGE     ?= postgres:16-alpine
PG_CONTAINER ?= webitel-wfm-test-pg

# The leader election tests (infra/leader) skip unless WFM_TEST_CONSUL_ADDR is set; test-consul
# runs them against a throwaway consul agent.

CONSUL_IMAGE     ?= hashicorp/consul:1.20
CONSUL_CONTAINER ?= webitel-wfm-test-consul

.PHONY: test-pg test-consul

test-consul:
	@docker rm -f $(CONSUL_CONTAINER) >/dev/null 2>&1; \
	  docker run -d --rm --name $(CONSUL_CONTAINER) -p 127.0.0.1::8500 $(CONSUL_IMAGE) agent -dev -client=0.0.0.0 >/dev/null
	@trap 'docker rm -f $(CONSUL_CONTAINER) >/dev/null' EXIT; \
	  addr=$$(docker port $(CONSUL_CONTAINER) 8500/tcp); \
	  for i in $$(seq 30); do curl -sf http://$$addr/v1/status/leader | grep -q : && break; sleep 1; done; \
	  WFM_TEST_CONSUL_ADDR=$$addr go test -race -count=1 ./infra/leader/

test-pg:
	@docker rm -f $(PG_CONTAINER) >/dev/null 2>&1; \
	  docker run -d --rm --name $(PG_CONTAINER) -e POSTGRES_HOST_AUTH_METHOD=trust \
	  -p 127.0.0.1::5432 $(PG_IMAGE) -c max_connections=200 >/dev/null
	@trap 'docker rm -f $(PG_CONTAINER) >/dev/null' EXIT; \
	  for i in $$(seq 30); do docker exec $(PG_CONTAINER) pg_isready -U postgres -q && break; sleep 1; done; \
	  WFM_TEST_POSTGRES_DSN="postgres://postgres@127.0.0.1:$$(docker port $(PG_CONTAINER) 5432/tcp | sed 's/.*://')/postgres?sslmode=disable" \
	  go test -race -count=1 ./...
