SHA8 = $(shell git rev-parse --short=8 main)

build:
	docker login
	KO_DOCKER_REPO="docker.io/chremoas" GOFLAGS="-ldflags=-X=main.buildCommit=${SHA8} -mod=vendor" ko resolve --platform=linux/amd64 --tags ${SHA8},latest-dev -BRf config/ > release.yaml

tidy:
	go mod tidy
	go mod vendor

migrate:
	migrate -source file://sql/ --database postgres://chremoas_dev@10.42.1.30:5432/chremoas_dev2 up

migrate-drop:
	migrate -source file://sql/ --database postgres://chremoas_dev@10.42.1.30:5432/chremoas_dev2 drop

docker-local:
	KO_DOCKER_REPO=ko.local GOFLAGS="-ldflags=-X=main.buildCommit=$SHA8 -mod=vendor" ko resolve --platform=linux/amd64 --tags ${SHA8},dev-latest -Bf config/docker-compose.yaml | egrep -v "^$|\---" > docker-compose.yaml
