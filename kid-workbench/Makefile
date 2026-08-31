.PHONY: up down logs ps seed

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps

seed:
	docker compose run --rm seed

seed-demo:
	docker compose run --rm --entrypoint ./seed seed -mode=demo -days=60
