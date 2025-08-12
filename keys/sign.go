package keys

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
)

func Sign(privKey *ecdsa.PrivateKey, msg string) ([]byte, error) {
	msgHash := crypto.Keccak256Hash([]byte(msg)) // simple keccak256 of the message

	// Sign the hash with the private key.
	sig, err := crypto.Sign(msgHash.Bytes(), privKey)
	if err != nil {
		return nil, err
	}
	return sig, nil
}
