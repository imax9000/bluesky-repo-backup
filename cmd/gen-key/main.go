package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multicodec"
)

func main() {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate a new key: %s", err)
	}

	public := elliptic.MarshalCompressed(elliptic.P256(),
		private.Public().(*ecdsa.PublicKey).X,
		private.Public().(*ecdsa.PublicKey).Y)
	prefix := binary.AppendUvarint(nil, uint64(multicodec.P256Pub))

	publicEncoded, err := multibase.Encode(multibase.Base58BTC, append(prefix, public...))
	if err != nil {
		log.Fatalf("Failed to convert public key into text format: %s", err)
	}

	if err := os.WriteFile("key.pub", []byte(fmt.Sprintf("did:key:%s", publicEncoded)), 0644); err != nil {
		log.Fatalf("Failed to write public key: %s", err)
	}

	privateEncoded, err := x509.MarshalECPrivateKey(private)
	if err != nil {
		log.Fatalf("Failed to serialize private key: %s", err)
	}

	if err := os.WriteFile("key.priv", privateEncoded, 0600); err != nil {
		log.Fatalf("Failed to write private key: %s", err)
	}
}
