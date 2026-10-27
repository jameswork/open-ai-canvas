docker compose --env-file .env -f docker-compose.deploy.yml -f docker-compose.build.yml --progress=plain build backend web \
--build-arg https_proxy=http://172.17.0.92:7890
