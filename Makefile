tidy:
	go mod tidy
	go mod vendor

migrate:
	migrate -source file://sql/ --database postgres://chremoas_dev@10.42.1.30:5432/chremoas_dev_roles up

migrate-drop:
	migrate -source file://sql/ --database postgres://chremoas_dev@10.42.1.30:5432/chremoas_dev_roles drop