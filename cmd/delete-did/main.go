package main

import (
	"bytes"
	"cmp"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"

	"github.com/imax9000/bluesky-repo-backup/plc"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	plcAddr := os.Getenv("ATP_PLC_ADDR")
	if plcAddr == "" {
		plcAddr = "https://plc.directory"
	}

	_, private, err := readKeys()
	if err != nil {
		log.Fatalf("Failed to read keys: %s", err)
	}

	b, err := os.ReadFile("did.txt")
	if err != nil {
		log.Fatalf("Failed to read DID: %s", err)
	}
	did := string(b)

	prev, err := getLastOpCID(plcAddr, did)
	if err != nil {
		log.Fatalf("Failed to get CID of the last valid operation: %s", err)
	}

	op := &plc.Tombstone{
		Type: "plc_tombstone",
		Prev: prev,
	}

	buf := bytes.NewBuffer(nil)
	if err := op.MarshalCBOR(buf); err != nil {
		log.Fatalf("Failed to serialized unsigned op: %s", err)
	}

	unsignedHash := sha256.Sum256(buf.Bytes())
	sigR, sigS, err := ecdsa.Sign(rand.Reader, private, unsignedHash[:])
	if err != nil {
		log.Fatalf("Failed to sign the op: %s", err)
	}
	sig := append(sigR.FillBytes(make([]byte, 32)), sigS.FillBytes(make([]byte, 32))...)
	op.Sig = ptr(base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sig))

	payload, err := json.Marshal(op)
	if err != nil {
		log.Fatalf("Failed to marshal the op as JSON: %s", err)
	}

	log.Printf("%+v", op)

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

func getLastOpCID(plcAddr string, did string) (string, error) {
	type LogEntry struct {
		CreatedAt string `json:"createdAt"`
		Nullified bool   `json:"nullified"`
		CID       string `json:"cid"`
	}

	resp, err := http.Get(fmt.Sprintf("%s/%s/log/audit", plcAddr, did))
	if err != nil {
		return "", fmt.Errorf("failed to fetch the audit log: %w", err)
	}
	defer resp.Body.Close()

	entries := []LogEntry{}

	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return "", fmt.Errorf("decoding audit log as JSON: %w", err)
	}

	slices.SortFunc(entries, func(a LogEntry, b LogEntry) int { return -cmp.Compare(a.CreatedAt, b.CreatedAt) })
	for _, e := range entries {
		if e.Nullified {
			continue
		}
		return e.CID, nil
	}
	return "", fmt.Errorf("did not find any valid operations")
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
