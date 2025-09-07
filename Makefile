.PHONY: up down logs bye

up:
	docker compose up --build

upd:
	docker compose up -d --build

down:
	docker compose down 

logs:
	docker compose logs -f emsub

bye:
	docker compose down -v

