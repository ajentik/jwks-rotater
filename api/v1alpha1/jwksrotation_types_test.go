package v1alpha1

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestJWKSRotationSpec_Validation(t *testing.T) {
	tests := []struct {
		name    string
		spec    JWKSRotationSpec
		wantErr string
	}{
		{
			name: "valid RSA 2048",
			spec: JWKSRotationSpec{
				KeyType:          RSA,
				KeySize:          2048,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
				TargetSecret:     TargetSecretRef{Name: "my-secret"},
			},
		},
		{
			name: "valid RSA 4096",
			spec: JWKSRotationSpec{
				KeyType:          RSA,
				KeySize:          4096,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
				TargetSecret:     TargetSecretRef{Name: "my-secret"},
			},
		},
		{
			name: "valid ECDSA P-256",
			spec: JWKSRotationSpec{
				KeyType:          ECDSA,
				KeySize:          256,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
				TargetSecret:     TargetSecretRef{Name: "my-secret"},
			},
		},
		{
			name: "valid ECDSA P-384",
			spec: JWKSRotationSpec{
				KeyType:          ECDSA,
				KeySize:          384,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
				TargetSecret:     TargetSecretRef{Name: "my-secret"},
			},
		},
		{
			name: "invalid RSA key size",
			spec: JWKSRotationSpec{
				KeyType:          RSA,
				KeySize:          1024,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
				TargetSecret:     TargetSecretRef{Name: "my-secret"},
			},
			wantErr: "invalid key size 1024 for RSA",
		},
		{
			name: "invalid ECDSA key size",
			spec: JWKSRotationSpec{
				KeyType:          ECDSA,
				KeySize:          512,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
				TargetSecret:     TargetSecretRef{Name: "my-secret"},
			},
			wantErr: "invalid key size 512 for ECDSA",
		},
		{
			name: "retention must be greater than rotation",
			spec: JWKSRotationSpec{
				KeyType:          RSA,
				KeySize:          2048,
				RotationInterval: metav1.Duration{Duration: 72 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 24 * time.Hour},
				TargetSecret:     TargetSecretRef{Name: "my-secret"},
			},
			wantErr: "retentionPeriod must be greater than rotationInterval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

func TestJWKSRotationSpec_DefaultKeySize(t *testing.T) {
	tests := []struct {
		name     string
		keyType  KeyType
		wantSize int
	}{
		{"RSA defaults to 2048", RSA, 2048},
		{"ECDSA defaults to 256", ECDSA, 256},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := JWKSRotationSpec{
				KeyType:          tt.keyType,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
				TargetSecret:     TargetSecretRef{Name: "test"},
			}
			if err := spec.Validate(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if spec.KeySize != tt.wantSize {
				t.Errorf("KeySize = %d, want %d", spec.KeySize, tt.wantSize)
			}
		})
	}
}

func TestJWKSRotationPolicySpec_Validation(t *testing.T) {
	tests := []struct {
		name    string
		spec    JWKSRotationPolicySpec
		wantErr string
	}{
		{
			name: "valid RSA policy",
			spec: JWKSRotationPolicySpec{
				KeyType:          RSA,
				KeySize:          2048,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
			},
		},
		{
			name: "ECDSA defaults to 256",
			spec: JWKSRotationPolicySpec{
				KeyType:          ECDSA,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
			},
		},
		{
			name: "invalid key size for ECDSA",
			spec: JWKSRotationPolicySpec{
				KeyType:          ECDSA,
				KeySize:          2048,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
			},
			wantErr: "invalid key size 2048 for ECDSA",
		},
		{
			name: "retention must exceed rotation",
			spec: JWKSRotationPolicySpec{
				KeyType:          RSA,
				KeySize:          2048,
				RotationInterval: metav1.Duration{Duration: 72 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 24 * time.Hour},
			},
			wantErr: "retentionPeriod must be greater than rotationInterval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

func TestJWKSRotationSpec_PublicSecretName(t *testing.T) {
	spec := JWKSRotationSpec{
		TargetSecret: TargetSecretRef{Name: "auth-jwks"},
	}
	got := spec.PublicSecretName()
	want := "auth-jwks-public"
	if got != want {
		t.Errorf("PublicSecretName() = %q, want %q", got, want)
	}
}
