SHA8 = $(shell git rev-parse --short=8 main)
VERSION = $(shell cat VERSION)

build:
	docker login
	KO_DOCKER_REPO="docker.io/chremoas" GOFLAGS="-ldflags=-X=main.buildCommit=${SHA8} -mod=vendor" ko resolve --platform=linux/amd64 --tags ${SHA8},${VERSION},latest -BRf config/ > release.yaml
	git tag ${VERSION}
	git push origin ${VERSION}

tidy:
	go mod tidy
	go mod vendor

migrate:
	migrate -source file://sql/ --database postgres://chremoas_aba@10.42.1.30:5432/chremoas_aba2 up

migrate-drop:
	migrate -source file://sql/ --database postgres://chremoas_dev@10.42.1.30:5432/chremoas_dev drop

docker-local:
	KO_DOCKER_REPO=ko.local GOFLAGS="-ldflags=-X=main.buildCommit=$SHA8 -mod=vendor" ko resolve --platform=linux/amd64 --tags ${SHA8},dev-latest -Bf config/docker-compose.yaml | egrep -v "^$|\---" > docker-compose.yaml
