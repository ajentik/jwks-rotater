package jwks

import (
	"crypto"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-jose/go-jose/v4"
)

type keyEntry struct {
	jwk jose.JSONWebKey
	iat time.Time
}

// KeyStore manages a set of JWK keys with creation timestamps.
type KeyStore struct {
	keys []keyEntry
}

// NewKeyStore creates an empty KeyStore.
func NewKeyStore() *KeyStore {
	return &KeyStore{}
}

// AddKey adds a ManagedKey to the store.
func (ks *KeyStore) AddKey(k *ManagedKey) {
	jwk := jose.JSONWebKey{
		Key:   k.Key,
		KeyID: k.KeyID,
	}
	ks.keys = append(ks.keys, keyEntry{jwk: jwk, iat: k.CreatedAt})
}

// Len returns the number of keys in the store.
func (ks *KeyStore) Len() int {
	return len(ks.keys)
}

// Keys returns the managed keys with their metadata.
func (ks *KeyStore) Keys() []ManagedKey {
	result := make([]ManagedKey, len(ks.keys))
	for i, e := range ks.keys {
		result[i] = ManagedKey{
			Key:       e.jwk.Key,
			KeyID:     e.jwk.KeyID,
			CreatedAt: e.iat,
		}
	}
	return result
}

type jwksJSON struct {
	Keys []json.RawMessage `json:"keys"`
}

type jwkWithIAT struct {
	jose.JSONWebKey
	IAT int64 `json:"iat"`
}

// MarshalJSON serializes the keystore to a JWKS JSON document with iat fields.
func (ks *KeyStore) MarshalJSON() ([]byte, error) {
	var rawKeys []json.RawMessage
	for _, e := range ks.keys {
		data, err := e.jwk.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshaling key %s: %w", e.jwk.KeyID, err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("re-parsing key JSON: %w", err)
		}
		iatBytes, _ := json.Marshal(e.iat.Unix())
		m["iat"] = iatBytes
		merged, err := json.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("merging iat into key: %w", err)
		}
		rawKeys = append(rawKeys, merged)
	}
	return json.Marshal(jwksJSON{Keys: rawKeys})
}

// MarshalPublicJSON serializes only the public components of each key.
func (ks *KeyStore) MarshalPublicJSON() ([]byte, error) {
	var rawKeys []json.RawMessage
	for _, e := range ks.keys {
		pub := e.jwk.Public()
		data, err := pub.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshaling public key %s: %w", e.jwk.KeyID, err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("re-parsing public key JSON: %w", err)
		}
		iatBytes, _ := json.Marshal(e.iat.Unix())
		m["iat"] = iatBytes
		merged, err := json.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("merging iat into public key: %w", err)
		}
		rawKeys = append(rawKeys, merged)
	}
	return json.Marshal(jwksJSON{Keys: rawKeys})
}

// ParseKeyStore deserializes a JWKS JSON document back into a KeyStore.
func ParseKeyStore(data []byte) (*KeyStore, error) {
	var raw jwksJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing JWKS: %w", err)
	}

	ks := NewKeyStore()
	for _, rawKey := range raw.Keys {
		var jwk jose.JSONWebKey
		if err := jwk.UnmarshalJSON(rawKey); err != nil {
			return nil, fmt.Errorf("parsing JWK: %w", err)
		}

		var extra struct {
			IAT int64 `json:"iat"`
		}
		if err := json.Unmarshal(rawKey, &extra); err != nil {
			return nil, fmt.Errorf("parsing iat: %w", err)
		}

		iat := time.Unix(extra.IAT, 0).UTC()
		ks.keys = append(ks.keys, keyEntry{jwk: jwk, iat: iat})
	}
	return ks, nil
}

// RemoveExpired removes keys older than the retention period.
// It never removes the most recent key. Returns removed key IDs.
func (ks *KeyStore) RemoveExpired(retention time.Duration, now time.Time) []string {
	if len(ks.keys) <= 1 {
		return nil
	}

	sort.Slice(ks.keys, func(i, j int) bool {
		return ks.keys[i].iat.Before(ks.keys[j].iat)
	})

	cutoff := now.Add(-retention)
	var kept []keyEntry
	var removed []string

	for i, e := range ks.keys {
		isNewest := i == len(ks.keys)-1
		if !isNewest && e.iat.Before(cutoff) {
			removed = append(removed, e.jwk.KeyID)
		} else {
			kept = append(kept, e)
		}
	}

	ks.keys = kept
	return removed
}

// OldestKeyAge returns the age of the oldest key relative to now.
func (ks *KeyStore) OldestKeyAge(now time.Time) time.Duration {
	if len(ks.keys) == 0 {
		return 0
	}
	oldest := ks.keys[0].iat
	for _, e := range ks.keys[1:] {
		if e.iat.Before(oldest) {
			oldest = e.iat
		}
	}
	return now.Sub(oldest)
}

// NewestKey returns the most recently created key, or nil if empty.
func (ks *KeyStore) NewestKey() *ManagedKey {
	if len(ks.keys) == 0 {
		return nil
	}
	newest := ks.keys[0]
	for _, e := range ks.keys[1:] {
		if e.iat.After(newest.iat) {
			newest = e
		}
	}
	return &ManagedKey{
		Key:       newest.jwk.Key.(crypto.PrivateKey),
		KeyID:     newest.jwk.KeyID,
		CreatedAt: newest.iat,
	}
}
