# GOPROXY=https://goproxy.cn,direct \
CANVAS_BACKEND_ADDR=127.0.0.1:8080 \
CANVAS_BACKEND_DATA_DIR=$(pwd)/../.local/project-workbench-debug \
GOCACHE=$(pwd)/../.local/cache/go-build \
GOMODCACHE=$(pwd)/../.local/cache/go-mod \
CANVAS_WHISPER_BASE_URL=http://127.0.0.1:8082 \
go run ./cmd/server
