# Makefile for building and running Docker containers for server and client

# Image names
SERVER_IMAGE = hashcash-server
CLIENT_IMAGE = hashcash-client

# Docker Compose file
DOCKER_COMPOSE = docker-compose.yml

# Default target
.PHONY: all
all: build run

# Build the server and client Docker images
.PHONY: build
build:
	@echo "Building server Docker image..."
	docker build -f Dockerfile.server -t $(SERVER_IMAGE) .
	@echo "Building client Docker image..."
	docker build -f Dockerfile.client -t $(CLIENT_IMAGE) .

# Run the server and client Docker images separately
.PHONY: run
run:
	@echo "Running the server container..."
	docker run -d --name server -p 8080:8080 $(SERVER_IMAGE)
	@echo "Running the client container..."
	docker run -d --name client -p 8081:8081 $(CLIENT_IMAGE)

# Stop the containers
.PHONY: stop
stop:
	@echo "Stopping and removing containers..."
	docker stop client server
	docker rm client server

# Build and run with Docker Compose
.PHONY: compose
compose:
	@echo "Starting up services with Docker Compose..."
	docker-compose -f $(DOCKER_COMPOSE) up -d

# Stop services with Docker Compose
.PHONY: compose-stop
compose-stop:
	@echo "Stopping services with Docker Compose..."
	docker-compose -f $(DOCKER_COMPOSE) down

# Build images using Docker Compose
.PHONY: compose-build
compose-build:
	@echo "Building images with Docker Compose..."
	docker-compose -f $(DOCKER_COMPOSE) build

# Show logs for server and client
.PHONY: logs
logs:
	@echo "Displaying logs for server..."
	docker logs server
	@echo "Displaying logs for client..."
	docker logs client

# Clean up unused Docker resources
.PHONY: clean
clean:
	@echo "Removing unused Docker images and volumes..."
	docker system prune -f
	docker volume prune -f
