package keys

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/ATMackay/eth-proxy/keys/data"
	"github.com/ethereum/go-ethereum/crypto"
)

const fixedMessage = "this is my fixed message to sign"

func Test_Deterministic_Signatures(t *testing.T) {
	var keys []keyRecord
	if err := readJSON(data.ETHTestKeys, &keys); err != nil {
		t.Fatalf("read json: %v", err)
	}

	var sigs []signatureRecord
	if err := readJSON(data.ETHTestSigs, &sigs); err != nil {
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

func readJSON(jSonStr string, v any) error {
	return json.Unmarshal([]byte(jSonStr), v)
}
