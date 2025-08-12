package keys

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

const fixedMessage = "this is my fixed message to sign"

func Test_Deterministic_Signatures(t *testing.T) {

	// 2) Save to a .json file (under test temp dir)
	keysJsonPath := filepath.Join("data/eth_keys.json")

	// 3) Reload and test signature generation for a fixed message
	var keys []keyRecord
	if err := readJSON(keysJsonPath, &keys); err != nil {
		t.Fatalf("read json: %v", err)
	}

	sigsJsonPath := filepath.Join("data/eth_sigs.json")

	var sigs []signatureRecord
	if err := readJSON(sigsJsonPath, &sigs); err != nil {
		t.Fatalf("read json: %v", err)
	}

	for i, r := range keys {
		priv, err := crypto.HexToECDSA(r.PrivateKeyHex)
		if err != nil {
			t.Fatalf("hex to ecdsa %d: %v", i, err)
		}
		// Sign the hash with the private key.
		sig, err := Sign(priv, fixedMessage)
		if err != nil {
			t.Fatalf("sign %d: %v", i, err)
		}

		// Verify signature using the corresponding public key.
		pubBytes := crypto.FromECDSAPub(&priv.PublicKey) // uncompressed 65-byte pubkey
		if ok := crypto.VerifySignature(pubBytes, crypto.Keccak256Hash([]byte(fixedMessage)).Bytes(), sig[:64]); !ok {
			t.Fatalf("verify %d: signature invalid", i)
		}

		// Ensure deterministic signature generation
		if sigs[i].AddressHex != r.AddressHex {
			t.Fatalf("deterministic signature %d: address mismatch", i)
		}

		if sigs[i].Signature != hex.EncodeToString(sig) {
			t.Fatalf("deterministic signature %d: mismatch", i)
		}
	}

}

func readJSON(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close file: %v", err)
		}
	}()

	if err := json.NewDecoder(f).Decode(&v); err != nil {
		return err
	}
	return nil
}
