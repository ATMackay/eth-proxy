package keys

type keyRecord struct {
	PrivateKeyHex string `json:"private_key"`
	AddressHex    string `json:"address"`
}

type signatureRecord struct {
	AddressHex string `json:"address"`
	Signature  string `json:"signature"`
}
