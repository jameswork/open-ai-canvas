docker compose --env-file .env -f docker-compose.deploy.yml -f docker-compose.build.yml  -f docker-compose.expose.yml up -d --wait --wait-timeout 600 backend web
