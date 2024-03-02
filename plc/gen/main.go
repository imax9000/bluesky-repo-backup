package main

import (
	"log"

	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/imax9000/bluesky-repo-backup/plc"
)

func main() {
	if err := typegen.WriteMapEncodersToFile("cbor_gen.go", "plc", plc.Service{}, plc.Op{}, plc.Tombstone{}); err != nil {
		log.Fatalf("%s", err)
	}
}
