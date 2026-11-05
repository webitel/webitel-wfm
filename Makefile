# The real-Postgres tests (infra/storage/dbsql) skip unless WFM_TEST_POSTGRES_DSN is set.
# test-pg starts a throwaway Postgres in docker, runs the whole suite against it, and removes it.

PG_IMAGE     ?= postgres:16-alpine
PG_CONTAINER ?= webitel-wfm-test-pg

.PHONY: test-pg

test-pg:
	@docker rm -f $(PG_CONTAINER) >/dev/null 2>&1; \
	  docker run -d --rm --name $(PG_CONTAINER) -e POSTGRES_HOST_AUTH_METHOD=trust \
	  -p 127.0.0.1::5432 $(PG_IMAGE) -c max_connections=200 >/dev/null
	@trap 'docker rm -f $(PG_CONTAINER) >/dev/null' EXIT; \
	  for i in $$(seq 30); do docker exec $(PG_CONTAINER) pg_isready -U postgres -q && break; sleep 1; done; \
	  WFM_TEST_POSTGRES_DSN="postgres://postgres@127.0.0.1:$$(docker port $(PG_CONTAINER) 5432/tcp | sed 's/.*://')/postgres?sslmode=disable" \
	  go test -race -count=1 ./...
