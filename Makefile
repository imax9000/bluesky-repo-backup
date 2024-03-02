.PHONY: all create-did

all: | key.pub key.priv
	@echo "PUBLIC_KEY=$$(cat key.pub)"
	@echo "PRIVATE_KEY=$$(hexdump -v -e '/1 "%02x"' key.priv)"

key.pub key.priv:
	go run ./cmd/gen-key

create-did: | key.pub key.priv
	go run ./cmd/create-did

delete-did: did.txt | key.pub key.priv
	go run ./cmd/delete-did

add-key: | key.pub key.priv
	go run ./cmd/add-key-to-plc
