package cmd

import (
	"encoding/json"
	"path/filepath"
	"fmt"
	"os"

	beinternal "github.com/HarishChandran3304/better-env/internal"
	"github.com/ProtonMail/gopenpgp/v3/crypto"
)

type ProjectConfig struct {
	Keys []string `json:"keys,omitempty"`
	Local map[string]interface{} `json:"local,omitempty"`
}

func ParseConfig() ([]string, error) {
	var pc ProjectConfig
	var secrets []string

	configPath := filepath.Join(".", ".better-env")
	if !exists(configPath) {
		return secrets, fmt.Errorf("failed to get the filepath")
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return secrets, fmt.Errorf("failed to get configdata need to run init")
	}

	if err := json.Unmarshal(configData, &pc); err != nil {
		return secrets, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	s, err := parseSecrets(pc); 
	if err != nil {
		return secrets, fmt.Errorf("failed to parse secrets: %w", err)
	}
	secrets = append(secrets, s...)


	l, err := parseLocal(pc); 
	if err != nil {
		return secrets, fmt.Errorf("failed to parse local env: %w", err)
	}
	secrets = append(secrets, l...)

	return secrets, nil
}

func parseLocal(pc ProjectConfig) ([]string, error) {
	env := []string{}
	for key, value := range pc.Local {
		var strValue string
		switch v := value.(type) {
		case string:
			strValue = v
		case float64:
			strValue = fmt.Sprintf("%.0f", v)
		case int:
			strValue = fmt.Sprintf("%d", v)
		case bool:
			strValue = fmt.Sprintf("%t", v)
		default:
			strValue = fmt.Sprintf("%v", v)
		}
		env = append(env, fmt.Sprintf("%s=%s", key, strValue))
	}
	return env, nil
}

func parseSecrets(pc ProjectConfig) ([]string, error) {
	storePath, err := getStorePath()
	if err != nil {
		return []string{}, fmt.Errorf("failed to get store path: %w", err)
	}
	privateKeyPath := filepath.Join(storePath, "private.key")
	privateKeyArmored, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return []string{}, fmt.Errorf("failed to get the private key: %w", err)
	}

	privateKey, err := beinternal.PromptAndUnlockPrivateKey(privateKeyArmored, true, 3)
	if err != nil {
		return []string{}, fmt.Errorf("failed to unlock the private key: %w", err)
	}
	defer privateKey.ClearPrivateParams()

	secretsPath := filepath.Join(storePath, SecretsFileName)
	encryptedData, err := os.ReadFile(secretsPath)
	if err != nil {
		return []string{}, fmt.Errorf("failed to get secrets file: %w", err)
	}

	pgp := crypto.PGP()
	decHandle, err := pgp.Decryption().DecryptionKey(privateKey).New()
	if err != nil {
		return []string{}, fmt.Errorf("failed to create decryption handle: %w", err)
	}
	defer decHandle.ClearPrivateParams()

	decrypted, err := decHandle.Decrypt(encryptedData, crypto.Bytes)
	if err != nil {
		return []string{}, fmt.Errorf("failed to decrypt data: %w", err)
	}

	var secrets map[string]string
	if err := json.Unmarshal(decrypted.Bytes(), &secrets); err != nil {
		return []string{}, fmt.Errorf("failed to unmarshal decrypted data: %w", err)
	}

	env := os.Environ()
	
	// Standard secrets
	if len(pc.Keys) == 0 {
		for key, val := range secrets {
			env = append(env, fmt.Sprintf("%s=%s", key, val))
		}
	} else {
		for _, key := range pc.Keys {
			if value, exists := secrets[key]; exists {
				env = append(env, fmt.Sprintf("%s=%s", key, value))
			} else {
				fmt.Fprintf(os.Stderr, "warning: '%s' not found in secrets store\n", key)
			}
		}
	}

	return env, nil
}
