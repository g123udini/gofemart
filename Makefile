include .env
$(shell touch .env.local)
include .env.local

export $(shell sed 's/=.*//' .env)
export $(shell sed 's/=.*//' .env.local)

DOCKER_COMPOSE?=docker compose
#определяем запуск из под раннера
ifndef CI
	DOCKER_COMPOSE_CONFIG := -f docker-compose.yml -f docker-compose.yml
else
	DOCKER_COMPOSE_CONFIG := -f docker-compose.ci.yml
endif

ifneq ("$(wildcard docker-compose.override.yml)","")
    DOCKER_COMPOSE_CONFIG += -f docker-compose.override.yml
endif

pull: ## Обновить образа
	$(DOCKER_COMPOSE) $(DOCKER_COMPOSE_CONFIG) pull

up: ## Поднять контейнеры
	$(DOCKER_COMPOSE) $(DOCKER_COMPOSE_CONFIG) up -d --remove-orphans

in-db: ## Войти в mysql контейнер
	$(DOCKER_COMPOSE) $(DOCKER_COMPOSE_CONFIG) exec postgres bash

stop: ## Остановить контейнеры
	$(DOCKER_COMPOSE) $(DOCKER_COMPOSE_CONFIG) stop
