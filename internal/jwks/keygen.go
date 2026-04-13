package jwks

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
)

// ManagedKey holds a generated key with its metadata.
type ManagedKey struct {
	Key       crypto.PrivateKey
	KeyID     string
	CreatedAt time.Time
}

// GenerateKey creates a new cryptographic key pair with a unique kid and creation timestamp.
func GenerateKey(keyType string, keySize int) (*ManagedKey, error) {
	var privKey crypto.PrivateKey
	var pubKey crypto.PublicKey

	switch keyType {
	case "RSA":
		if keySize != 2048 && keySize != 4096 {
			return nil, fmt.Errorf("invalid RSA key size: %d (must be 2048 or 4096)", keySize)
		}
		k, err := rsa.GenerateKey(rand.Reader, keySize)
		if err != nil {
			return nil, fmt.Errorf("generating RSA key: %w", err)
		}
		privKey = k
		pubKey = &k.PublicKey
	case "ECDSA":
		var curve elliptic.Curve
		switch keySize {
		case 256:
			curve = elliptic.P256()
		case 384:
			curve = elliptic.P384()
		default:
			return nil, fmt.Errorf("invalid ECDSA key size: %d (must be 256 or 384)", keySize)
		}
		k, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generating ECDSA key: %w", err)
		}
		privKey = k
		pubKey = &k.PublicKey
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	jwk := jose.JSONWebKey{Key: pubKey}
	kid, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("computing key thumbprint: %w", err)
	}

	return &ManagedKey{
		Key:       privKey,
		KeyID:     base64.RawURLEncoding.EncodeToString(kid),
		CreatedAt: time.Now().UTC(),
	}, nil
}
