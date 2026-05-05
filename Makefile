.PHONY: up down ps logs smoke

COMPOSE := docker compose -f compose.yaml

up:
	DOCKER_BUILDKIT=0 COMPOSE_DOCKER_CLI_BUILD=0 $(COMPOSE) up -d --build

down:
	$(COMPOSE) down

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f --tail=100

smoke:
	curl -fsS http://localhost:8080/healthz
	curl -sS -i -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" --data '{"login":"vadim","password":"secret"}'
