package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"

	"github.com/bluesky-social/indigo/xrpc"
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
	"github.com/uabluerail/bsky-tools/xrpcauth"

	"github.com/imax9000/bluesky-repo-backup/plc"
)

type Config struct {
	DID       string
	Password  string
	PLCAddr   string `envconfig:"ATP_PLC_ADDR" default:"https://plc.directory"`
	PublicKey string `envconfig:"PUBLIC_KEY"`
}

var config Config

func main() {
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("envconfig.Process: %s", err)
	}

	if config.PublicKey == "" {
		log.Fatalf("No public key provided in PUBLIC_KEY env variable")
	}

	resp, err := http.Get(fmt.Sprintf("%s/%s/data", config.PLCAddr, config.DID))
	if err != nil {
		log.Fatalf("Failed to fetch the current PLC data: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code from PLC: %s", resp.Status)
	}
	data := &plc.Op{}
	if err := json.NewDecoder(resp.Body).Decode(data); err != nil {
		log.Fatalf("Failed to unmarshal the response from PLC: %s", err)
	}

	if len(data.RotationKeys) > 0 && data.RotationKeys[0] == config.PublicKey {
		// All good, nothing to do.
		return
	}

	data.Type = "plc_operation"
	data.RotationKeys = append([]string{config.PublicKey},
		slices.DeleteFunc(data.RotationKeys, func(s string) bool { return s == config.PublicKey })...)
	data.Sig = ptr("invalid, all rotation keys are held hostage")

	pds := ""
	if data.Services != nil {
		pds = data.Services["atproto_pds"].Endpoint
	}
	if pds == "" {
		log.Fatalf("Did not find a PDS endpoint in data returned by PLC")
	}

	ctx := context.Background()
	client := xrpcauth.NewClientWithTokenSource(ctx, xrpcauth.PasswordAuth(config.DID, config.Password))
	client.Host = pds

	// Fails with "error calling MarshalJSON for type *util.LexiconTypeDecoder: lexicon type decoder can only handle record fields"
	//
	// req := &comatproto.IdentitySubmitPlcOperation_Input{
	// 	Operation: &util.LexiconTypeDecoder{Val: update},
	// }
	// if err := comatproto.IdentitySubmitPlcOperation(ctx, client, req); err != nil {
	// 	log.Fatalf("Failed to update rotation keys in PLC via PDS: %s", err)
	// }

	log.Printf("Sending the following operation to %s:\n%+v", pds, data)

	err = client.Do(ctx, xrpc.Procedure, "application/json",
		"com.atproto.identity.submitPlcOperation", nil,
		map[string]any{"operation": data}, nil)
	if err != nil {
		log.Fatalf("Failed to update rotation keys in PLC via PDS: %s", err)
	}
}

func ptr[T any](v T) *T {
	return &v
}
