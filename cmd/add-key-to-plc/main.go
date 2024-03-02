package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"

	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
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

	token := os.Getenv("TOKEN")
	// TODO: if our key is present in the data from PLC - use it for signing directly instead of
	// requesting the PDS to sign.
	if token == "" {
		err := comatproto.IdentityRequestPlcOperationSignature(ctx, client)
		if err != nil {
			log.Fatalf("Failed to request PLC operation signature from PDS: %s", err)
		}
		fmt.Println("Please check your email for the authorization code and add it to your .env file as a TOKEN variable")
		return
	}

	update := &comatproto.IdentitySignPlcOperation_Input{
		RotationKeys: append([]string{config.PublicKey},
			slices.DeleteFunc(data.RotationKeys, func(s string) bool { return s == config.PublicKey })...),
		Token: ptr(token),
	}

	// Fails with `decoding xrpc response: unrecognized lexicon type: ""`
	//
	// signedResp, err := comatproto.IdentitySignPlcOperation(ctx, client, update)
	// if err != nil {
	// 	log.Fatalf("Failed to get a signature for the PLC operation from PDS: %s", err)
	// }

	var signedOp struct {
		Operation plc.Op `json:"operation"`
	}
	err = client.Do(ctx, xrpc.Procedure, "application/json",
		"com.atproto.identity.signPlcOperation", nil, update, &signedOp)
	if err != nil {
		log.Fatalf("Failed to get a signature for the PLC operation from PDS: %s", err)
	}

	// Fails with "error calling MarshalJSON for type *util.LexiconTypeDecoder: lexicon type decoder can only handle record fields"
	//
	// req := &comatproto.IdentitySubmitPlcOperation_Input{
	// 	Operation: &util.LexiconTypeDecoder{Val: update},
	// }
	// if err := comatproto.IdentitySubmitPlcOperation(ctx, client, req); err != nil {
	// 	log.Fatalf("Failed to update rotation keys in PLC via PDS: %s", err)
	// }

	// Fails with "XRPC ERROR 400: InvalidRequest: Rotation keys do not include server's rotation key"
	//
	// err = client.Do(ctx, xrpc.Procedure, "application/json",
	// 	"com.atproto.identity.submitPlcOperation", nil,
	// 	signedOp, nil)
	// if err != nil {
	// 	log.Fatalf("Failed to update rotation keys in PLC via PDS: %s", err)
	// }

	log.Printf("Sending the following operation to PLC:\n%+v", signedOp)

	payload, err := json.Marshal(signedOp.Operation)
	if err != nil {
		log.Fatalf("Failed to marshal the signed operation as JSON: %s", err)
	}

	resp, err = http.Post(fmt.Sprintf("%s/%s", config.PLCAddr, config.DID), "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Fatalf("Failed to send the update request to PLC: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected response code from PLC: %s", resp.Status)
	}
	io.Copy(os.Stdout, resp.Body)
}

func ptr[T any](v T) *T {
	return &v
}
