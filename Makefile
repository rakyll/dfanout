proto:
    protoc --go_out=paths=source_relative:. --twirp_out=paths=source_relative:. proto/service.proto

dev:
	go run cmd/dfanout/*.go

docker-build:
	docker build --platform linux/amd64 -t dfanout .

docker-run:
	docker run -it -p 8080:8080/tcp -p 5432:5432/tcp dfanout

docker-bash:
	docker run -it --entrypoint /bin/bash dfanout

docker-push-google:
	docker tag dfanout:latest gcr.io/dfanout/dfanout:latest
	docker push gcr.io/dfanout/dfanout:latest
