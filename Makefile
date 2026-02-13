.PHONY: help install deploy-sepolia deploy-base run-backend clean

help:
	@echo "Token Points System - Makefile Commands"
	@echo ""
	@echo "Setup:"
	@echo "  install          Install all dependencies"
	@echo "  init-db          Initialize database"
	@echo ""
	@echo "Contracts:"
	@echo "  compile          Compile smart contracts"
	@echo "  deploy-sepolia   Deploy to Sepolia testnet"
	@echo "  deploy-base      Deploy to Base Sepolia testnet"
	@echo ""
	@echo "Run:"
	@echo "  run-backend      Start Go backend server (includes frontend)"
	@echo ""
	@echo "Clean:"
	@echo "  clean            Clean build artifacts"

install:
	cd contracts && npm install
	cd backend && go mod download

init-db:
	mysql -u root -p < database/schema.sql

compile:
	cd contracts && npm run compile

deploy-sepolia:
	cd contracts && npm run deploy:sepolia

deploy-base:
	cd contracts && npm run deploy:base-sepolia

run-backend:
	cd backend && go run cmd/main.go

clean:
	rm -rf contracts/cache contracts/artifacts contracts/node_modules
	rm -rf backend/tmp
	find . -name "*.log" -delete
