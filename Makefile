# M1 dev shortcuts. See docs/10-roadmap/milestones.md for scope; this repo
# is otherwise documentation-first (adr/, docs/, schemas/, playbooks/) with
# no build step of its own — validation/*.sh covers that content.

COMPOSE := docker compose -f deploy/compose/docker-compose.core.yml --profile core
DATABASE_URL ?= postgres://soar:soar_lab_only@127.0.0.1:5432/soar?sslmode=disable

.PHONY: db-up db-down migrate api web dev test test-integration validate

db-up:
	$(COMPOSE) up -d postgres

db-down:
	$(COMPOSE) down

migrate:
	cd services/api && atlas migrate apply --dir file://migrations --url "$(DATABASE_URL)"

api:
	cd services/api && DATABASE_URL="$(DATABASE_URL)" go run .

web:
	cd web && npm run dev

test:
	cd services/api && go test ./...

test-integration:
	cd services/api && go test -tags=integration ./integration/...

validate:
	bash validation/validate-json.sh
	bash validation/validate-playbooks.sh
	bash validation/validate-diagrams.sh
	bash validation/validate-links.sh
	bash validation/validate-source-traceability.sh
