package jwks

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"testing"
	"time"
)

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name      string
		keyType   string
		keySize   int
		wantErr   bool
		checkFunc func(t *testing.T, k *ManagedKey)
	}{
		{
			name:    "RSA 2048",
			keyType: "RSA",
			keySize: 2048,
			checkFunc: func(t *testing.T, k *ManagedKey) {
				priv, ok := k.Key.(*rsa.PrivateKey)
				if !ok {
					t.Fatal("expected *rsa.PrivateKey")
				}
				if priv.N.BitLen() != 2048 {
					t.Errorf("key size = %d, want 2048", priv.N.BitLen())
				}
			},
		},
		{
			name:    "RSA 4096",
			keyType: "RSA",
			keySize: 4096,
			checkFunc: func(t *testing.T, k *ManagedKey) {
				priv, ok := k.Key.(*rsa.PrivateKey)
				if !ok {
					t.Fatal("expected *rsa.PrivateKey")
				}
				if priv.N.BitLen() != 4096 {
					t.Errorf("key size = %d, want 4096", priv.N.BitLen())
				}
			},
		},
		{
			name:    "ECDSA P-256",
			keyType: "ECDSA",
			keySize: 256,
			checkFunc: func(t *testing.T, k *ManagedKey) {
				priv, ok := k.Key.(*ecdsa.PrivateKey)
				if !ok {
					t.Fatal("expected *ecdsa.PrivateKey")
				}
				if priv.Curve != elliptic.P256() {
					t.Error("expected P-256 curve")
				}
			},
		},
		{
			name:    "ECDSA P-384",
			keyType: "ECDSA",
			keySize: 384,
			checkFunc: func(t *testing.T, k *ManagedKey) {
				priv, ok := k.Key.(*ecdsa.PrivateKey)
				if !ok {
					t.Fatal("expected *ecdsa.PrivateKey")
				}
				if priv.Curve != elliptic.P384() {
					t.Error("expected P-384 curve")
				}
			},
		},
		{
			name:    "unsupported key type",
			keyType: "EdDSA",
			keySize: 256,
			wantErr: true,
		},
		{
			name:    "invalid RSA size",
			keyType: "RSA",
			keySize: 1024,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, err := GenerateKey(tt.keyType, tt.keySize)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if k.KeyID == "" {
				t.Error("kid must not be empty")
			}
			if k.CreatedAt.IsZero() {
				t.Error("createdAt must not be zero")
			}
			if time.Since(k.CreatedAt) > 5*time.Second {
				t.Error("createdAt should be recent")
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, k)
			}
		})
	}
}

func TestGenerateKey_UniqueKids(t *testing.T) {
	k1, err := GenerateKey("RSA", 2048)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := GenerateKey("RSA", 2048)
	if err != nil {
		t.Fatal(err)
	}
	if k1.KeyID == k2.KeyID {
		t.Error("two generated keys must have different kids")
	}
}
