package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/imax9000/bluesky-repo-backup/plc"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	plcAddr := os.Getenv("ATP_PLC_ADDR")
	if plcAddr == "" {
		plcAddr = "https://plc.directory"
	}

	public, private, err := readKeys()
	if err != nil {
		log.Fatalf("Failed to read keys: %s", err)
	}

	op := &plc.Op{
		Type:         "plc_operation",
		RotationKeys: []string{public},
		AlsoKnownAs:  []string{"at://example.com"},
		Services: map[string]plc.Service{
			"atproto_pds": {
				Type:     "AtprotoPersonalDataServer",
				Endpoint: "https://example.com",
			}},
		VerificationMethods: map[string]string{
			"atproto": public,
		},
	}

	b := bytes.NewBuffer(nil)
	if err := op.MarshalCBOR(b); err != nil {
		log.Fatalf("Failed to serialized unsigned op: %s", err)
	}

	unsignedHash := sha256.Sum256(b.Bytes())
	sigR, sigS, err := ecdsa.Sign(rand.Reader, private, unsignedHash[:])
	if err != nil {
		log.Fatalf("Failed to sign the op: %s", err)
	}
	sig := append(sigR.FillBytes(make([]byte, 32)), sigS.FillBytes(make([]byte, 32))...)
	op.Sig = ptr(base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sig))
	// op.Sig = ptr(base64.URLEncoding.EncodeToString(sig))

	b = bytes.NewBuffer(nil)
	if err := op.MarshalCBOR(b); err != nil {
		log.Fatalf("Failed to serialized signed op: %s", err)
	}
	hash := sha256.Sum256(b.Bytes())
	did := "did:plc:" + strings.ToLower(base32.StdEncoding.EncodeToString(hash[:])[:24])
	if err := os.WriteFile("did.txt", []byte(did), 0644); err != nil {
		log.Fatalf("Failed to write DID to file: %s", err)
	}

	payload, err := json.Marshal(op)
	if err != nil {
		log.Fatalf("Failed to marshal the op as JSON: %s", err)
	}

	resp, err := http.Post(fmt.Sprintf("%s/%s", plcAddr, did), "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Fatalf("Failed to send the request: %s", err)
	}
	if resp.StatusCode == http.StatusOK {
		return
	}

	fmt.Fprintf(os.Stderr, "Response: %d %s\n", resp.StatusCode, resp.Status)
	io.Copy(os.Stderr, resp.Body)
	resp.Body.Close()
	os.Exit(1)
}

func readKeys() (string, *ecdsa.PrivateKey, error) {
	b, err := os.ReadFile("key.pub")
	if err != nil {
		return "", nil, fmt.Errorf("reading public key file: %w", err)
	}
	pubKey := string(b)

	b, err = os.ReadFile("key.priv")
	if err != nil {
		return "", nil, fmt.Errorf("reading private key file: %w", err)
	}

	privKey, err := x509.ParseECPrivateKey(b)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return pubKey, privKey, nil

}

func ptr[T any](v T) *T {
	return &v
}
