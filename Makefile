tidy:
	go mod tidy
	go mod vendor

migrate:
	migrate -source file://sql/ --database postgres://chremoas_aba@10.42.1.30:5432/chremoas_aba up

migrate-drop:
	migrate -source file://sql/ --database postgres://chremoas_aba@10.42.1.30:5432/chremoas_aba drop