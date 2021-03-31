package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/flyteorg/flyteadmin/pkg/auth/config"

	"github.com/flyteorg/flytestdlib/logger"

	"github.com/spf13/cobra"
)

const (
	TokenHashKeyLength   = 32
	CookieHashKeyLength  = 64
	CookieBlockKeyLength = 32
	rsaPEMType           = "RSA PRIVATE KEY"
)

// GetInitSecretsCommand creates a command to issue secrets to be used for Auth settings. It writes the secrets to the
// working directory. The expectation is that they are put in a location and made available to the serve command later.
// To configure where the serve command looks for secrets, update this config:
// secrets:
//	secrets-prefix: <my custom path>
func GetInitSecretsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init-secrets",
		Short: "Generates secrets needed for OpenIDC and OAuth2 providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			secrets, err := createSecrets()
			if err != nil {
				return err
			}

			d, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory. Error: %w", err)
			}

			return writeSecrets(ctx, secrets, d)
		},
	}
}

type SecretsSet struct {
	TokenHashKey              []byte
	TokenSigningRSAPrivateKey *rsa.PrivateKey
	CookieHashKey             []byte
	CookieBlockKey            []byte
}

func writeSecrets(ctx context.Context, secrets SecretsSet, path string) error {
	err := ioutil.WriteFile(filepath.Join(path, config.SecretTokenHash), []byte(base64.RawStdEncoding.EncodeToString(secrets.TokenHashKey)), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to persist token hash key. Error: %w", err)
	}

	logger.Infof(ctx, "wrote %v", config.SecretTokenHash)

	err = ioutil.WriteFile(filepath.Join(path, config.SecretCookieHashKey), []byte(base64.RawStdEncoding.EncodeToString(secrets.CookieHashKey)), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to persist cookie hash key. Error: %w", err)
	}

	logger.Infof(ctx, "wrote %v", config.SecretCookieHashKey)

	err = ioutil.WriteFile(filepath.Join(path, config.SecretCookieBlockKey), []byte(base64.RawStdEncoding.EncodeToString(secrets.CookieBlockKey)), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to persist cookie block key. Error: %w", err)
	}

	logger.Infof(ctx, "wrote %v", config.SecretCookieBlockKey)

	keyOut, err := os.OpenFile(config.SecretTokenSigningRSAKey, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open key.pem for writing: %w", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(secrets.TokenSigningRSAPrivateKey)
	if err := pem.Encode(keyOut, &pem.Block{Type: rsaPEMType, Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write data to key.pem: %w", err)
	}

	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("error closing key.pem: %w", err)
	}

	logger.Infof(ctx, "wrote %v", config.SecretTokenSigningRSAKey)

	return nil
}

func createSecrets() (SecretsSet, error) {
	secret := make([]byte, TokenHashKeyLength)
	_, err := rand.Read(secret)
	if err != nil {
		return SecretsSet{}, fmt.Errorf("failed to issue token hash key. Error: %w", err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return SecretsSet{}, fmt.Errorf("failed to issue token signing key. Error: %w", err)
	}

	cookieHashKey := make([]byte, CookieHashKeyLength)
	_, err = rand.Read(cookieHashKey)
	if err != nil {
		return SecretsSet{}, fmt.Errorf("failed to issue cookie hash key. Error: %w", err)
	}

	cookieBlockKey := make([]byte, CookieBlockKeyLength)
	_, err = rand.Read(cookieBlockKey)
	if err != nil {
		return SecretsSet{}, fmt.Errorf("failed to issue cookie block key. Error: %w", err)
	}

	return SecretsSet{
		TokenHashKey:              secret,
		TokenSigningRSAPrivateKey: privateKey,
		CookieHashKey:             cookieHashKey,
		CookieBlockKey:            cookieBlockKey,
	}, nil
}
