package jwks

import (
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"
)

func TestKeyStore_AddAndSerialize(t *testing.T) {
	ks := NewKeyStore()

	k, err := GenerateKey("RSA", 2048)
	if err != nil {
		t.Fatal(err)
	}
	ks.AddKey(k)

	if ks.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", ks.Len())
	}

	data, err := ks.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := raw["keys"]; !ok {
		t.Fatal("JSON must contain 'keys' field")
	}

	ks2, err := ParseKeyStore(data)
	if err != nil {
		t.Fatalf("ParseKeyStore: %v", err)
	}
	if ks2.Len() != 1 {
		t.Fatalf("round-trip Len() = %d, want 1", ks2.Len())
	}

	keys := ks2.Keys()
	if keys[0].KeyID != k.KeyID {
		t.Errorf("round-trip kid = %q, want %q", keys[0].KeyID, k.KeyID)
	}
}

func TestKeyStore_MultipleKeys(t *testing.T) {
	ks := NewKeyStore()

	for range 3 {
		k, err := GenerateKey("RSA", 2048)
		if err != nil {
			t.Fatal(err)
		}
		ks.AddKey(k)
	}

	if ks.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", ks.Len())
	}

	data, err := ks.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	ks2, err := ParseKeyStore(data)
	if err != nil {
		t.Fatal(err)
	}
	if ks2.Len() != 3 {
		t.Fatalf("round-trip Len() = %d, want 3", ks2.Len())
	}
}

func TestKeyStore_PublicOnly(t *testing.T) {
	ks := NewKeyStore()

	k, err := GenerateKey("RSA", 2048)
	if err != nil {
		t.Fatal(err)
	}
	ks.AddKey(k)

	pubData, err := ks.MarshalPublicJSON()
	if err != nil {
		t.Fatal(err)
	}

	pub, err := ParseKeyStore(pubData)
	if err != nil {
		t.Fatal(err)
	}
	if pub.Len() != 1 {
		t.Fatalf("public Len() = %d, want 1", pub.Len())
	}

	keys := pub.Keys()
	if _, ok := keys[0].Key.(*rsa.PublicKey); !ok {
		t.Error("public JWKS should contain only public keys")
	}
}

func TestKeyStore_RemoveExpired(t *testing.T) {
	tests := []struct {
		name          string
		keyAges       []time.Duration
		retention     time.Duration
		wantRemaining int
		wantRemoved   int
	}{
		{
			name:          "no expired keys",
			keyAges:       []time.Duration{1 * time.Hour, 2 * time.Hour},
			retention:     72 * time.Hour,
			wantRemaining: 2,
			wantRemoved:   0,
		},
		{
			name:          "one expired key",
			keyAges:       []time.Duration{1 * time.Hour, 100 * time.Hour},
			retention:     72 * time.Hour,
			wantRemaining: 1,
			wantRemoved:   1,
		},
		{
			name:          "all expired keeps newest",
			keyAges:       []time.Duration{100 * time.Hour, 200 * time.Hour},
			retention:     72 * time.Hour,
			wantRemaining: 1,
			wantRemoved:   1,
		},
		{
			name:          "single key never removed even if expired",
			keyAges:       []time.Duration{200 * time.Hour},
			retention:     72 * time.Hour,
			wantRemaining: 1,
			wantRemoved:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := NewKeyStore()
			now := time.Now().UTC()

			for _, age := range tt.keyAges {
				k, err := GenerateKey("RSA", 2048)
				if err != nil {
					t.Fatal(err)
				}
				k.CreatedAt = now.Add(-age)
				ks.AddKey(k)
			}

			removed := ks.RemoveExpired(tt.retention, now)

			if ks.Len() != tt.wantRemaining {
				t.Errorf("remaining = %d, want %d", ks.Len(), tt.wantRemaining)
			}
			if len(removed) != tt.wantRemoved {
				t.Errorf("removed = %d, want %d", len(removed), tt.wantRemoved)
			}
		})
	}
}

func TestKeyStore_OldestKeyAge(t *testing.T) {
	ks := NewKeyStore()
	now := time.Now().UTC()

	k1, _ := GenerateKey("RSA", 2048)
	k1.CreatedAt = now.Add(-48 * time.Hour)
	ks.AddKey(k1)

	k2, _ := GenerateKey("RSA", 2048)
	k2.CreatedAt = now.Add(-24 * time.Hour)
	ks.AddKey(k2)

	age := ks.OldestKeyAge(now)
	if age < 47*time.Hour || age > 49*time.Hour {
		t.Errorf("OldestKeyAge = %v, want ~48h", age)
	}
}
