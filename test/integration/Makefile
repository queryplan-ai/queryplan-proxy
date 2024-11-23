

.PNONY: build
build:
	go build -o bin/integration-test .

.PNONY: postgres
postgres: build
	./bin/integration-test postgres

.PHONY: mysql
mysql: build
	./bin/integration-test mysql
