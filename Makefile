SHELL := /bin/sh

.PHONY: test test-go test-sol demo-dry demo-live guardd

test: test-sol test-go

test-sol:
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; \
	forge test -vvv

test-go:
	@go test ./... -v | grep -v "\\[no test files\\]"

guardd:
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; \
	go run ./cmd/guardd

demo-dry:
	@echo "Will run:"
	@echo "  forge script script/CreateRequest.s.sol:CreateRequest --rpc-url $$RPC_URL --broadcast"
	@echo "  then start guardd in another terminal: make guardd"
	@echo ""
	@echo "Checks:"
	@if [ -z "$$RPC_URL" ]; then echo "RPC_URL missing"; exit 1; fi
	@if [ -z "$$CONTRACT_ADDRESS" ]; then echo "CONTRACT_ADDRESS missing"; exit 1; fi
	@echo "OK"

demo-live:
	@if [ "$$CONFIRM_LIVE" != "1" ]; then \
		echo "Refusing to broadcast. Run: CONFIRM_LIVE=1 make demo-live"; \
		exit 1; \
	fi
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; \
	forge script script/CreateRequest.s.sol:CreateRequest --rpc-url "$$RPC_URL" --broadcast
