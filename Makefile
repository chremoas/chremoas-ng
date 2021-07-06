SHA8 = $(shell git rev-parse --short=8 main)
SOURCE_DATE_EPOCH = $(shell date +%s)

tidy:
	go mod tidy
	go mod vendor

migrate:
	migrate -source file://sql/ --database postgres://chremoas_aba@10.42.1.30:5432/chremoas_aba up

migrate-drop:
	migrate -source file://sql/ --database postgres://chremoas_aba@10.42.1.30:5432/chremoas_aba drop

docker-local:
	KO_DOCKER_REPO=ko.local GOFLAGS="-ldflags=-X=main.buildCommit=$SHA8 -mod=vendor" ko publish ./ --platform=linux/amd64 --tags ${SHA8},dev-latest