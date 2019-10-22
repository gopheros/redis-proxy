
test:
	go test

build:
	go build -o redis-proxy

run:
	bash start.sh

clean:
	@echo "Cleaning up..."
	rm -rf redis-proxy
	docker-compose down
	docker rmi -f redis-proxy:1.0