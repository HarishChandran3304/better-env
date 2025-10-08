package cmd

import (
	"bufio"
	"bytes"
	"crypto"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	ConfigFileName  = "config.json"
	SecretsFileName = "secrets.gpg"
)

const Banner = `
    __         __  __                                 
   / /_  ___  / /_/ /____  _____            ___  ____ _   __
  / __ \/ _ \/ __/ __/ _ \/ ___/  ______   / _ \/ __ \ | / /
 / /_/ /  __/ /_/ /_/  __/ /     /_____/  /  __/ / / / |/ / 
/_.___/\___/\__/\__/\___/_/               \___/_/ /_/|___/`

type Config struct {
	StorePath      string    `json:"store_path"`
	KeyFingerprint string    `json:"key_fingerprint"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SetupCommand struct {
	reader     *bufio.Reader
	passphrase string
}

func NewSetupCommand() *SetupCommand {
	return &SetupCommand{
		reader:     bufio.NewReader(os.Stdin),
		passphrase: "",
	}
}

func (s *SetupCommand) Run() error {
	fmt.Println(Banner)
	fmt.Println()

	// Get the fixed store path
	storePath, err := getStorePath()
	if err != nil {
		return fmt.Errorf("failed to determine store path: %w", err)
	}

	configPath := filepath.Join(storePath, ConfigFileName)

	// Check if already configured
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("⚠️  better-env is already configured at: %s\n", storePath)
		if !s.askYesNo("Do you want to reconfigure?") {
			fmt.Println("Setup cancelled.")
			return nil
		}
	}

	// Step 0: Create store directory
	if err := os.MkdirAll(storePath, 0700); err != nil {
		return fmt.Errorf("failed to create store directory: %w", err)
	}

	fmt.Println()
	fmt.Println("🚀 better-env Setup")

	// Step 1: GPG Key Generation
	entity, err := s.handleKeySetup()
	if err != nil {
		return fmt.Errorf("key setup failed: %w", err)
	}

	// Step 2: Save configuration
	config := Config{
		StorePath:      storePath,
		KeyFingerprint: fmt.Sprintf("%X", entity.PrimaryKey.Fingerprint),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.saveConfig(storePath, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Step 3: Save the key pair
	if err := s.saveKeyPair(storePath, entity); err != nil {
		return fmt.Errorf("failed to save key pair: %w", err)
	}

	// Step 4: Initialize empty secrets file
	if err := s.initializeSecretsFile(storePath, entity); err != nil {
		return fmt.Errorf("failed to initialize secrets file: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ better-env setup complete!")
	fmt.Printf("📁 Store location: %s\n", storePath)
	fmt.Printf("🔑 Key fingerprint: %s\n", config.KeyFingerprint)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Enter this command to start using better-env:")
	// TODO: Add a command to set the alias and prompt the user to run it
	// We need an appropriate command they can run to set the alias for eval-ing the launch command
	// Something like: echo "alias bnv launch='eval \"$(bnv launch)\"'" >> ~/.bashrc and source ~/.bashrc
	// Basically since we cant set env vars to parent processes, we will output each key value pair in the form of EXPORT KEY=VALUE and the eval will set it for us
	fmt.Println("  2. Navigate to your project: cd my/project")
	fmt.Println("  3. Initialize better-env: bnv init")
	fmt.Println("  4. Add secrets: bnv set KEY VALUE")
	fmt.Println("  5. Load secrets: bnv launch")
	fmt.Println()
	// TODO: Add a link to the docs
	fmt.Println("Check out the docs at https://github.com/HarishChandran3304/better-env for more information!")

	return nil
}

// getStorePath returns the fixed store path for better-env
func getStorePath() (string, error) {
	// Use os.UserConfigDir() which handles cross-platform config directories
	// Linux/Unix: ~/.config
	// macOS: ~/Library/Application Support (but also supports ~/.config)
	// Windows: %APPDATA%
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "better-env"), nil
}

func (s *SetupCommand) handleKeySetup() (*openpgp.Entity, error) {
	fmt.Println()
	fmt.Println("🔑 First, let's set up a new GPG key for you.")
	return s.generateNewKey()
}

func (s *SetupCommand) generateNewKey() (*openpgp.Entity, error) {
	fmt.Print("Enter your name: ")
	name := s.readLine()
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	fmt.Print("Enter your email: ")
	email := s.readLine()
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	fmt.Print("Enter a passphrase: ")
	passphrase := s.readPassword()
	s.passphrase = passphrase

	config := &packet.Config{
		DefaultHash:            crypto.SHA256,
		DefaultCipher:          packet.CipherAES256,
		DefaultCompressionAlgo: packet.CompressionZLIB,
		RSABits:                2048,
	}

	entity, err := openpgp.NewEntity(name, "", email, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Note: self-signatures may be created lazily during serialization. We'll
	// handle potential re-signing safely inside saveKeyPair by temporarily
	// decrypting the key if needed.

	// Encrypt private key with passphrase if provided
	if passphrase != "" {
		if err := entity.PrivateKey.Encrypt([]byte(passphrase)); err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
	}

	return entity, nil
}

func (s *SetupCommand) saveConfig(storePath string, config Config) error {
	configPath := filepath.Join(storePath, ConfigFileName)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func (s *SetupCommand) saveKeyPair(storePath string, entity *openpgp.Entity) error {
	// Save private key
	privateKeyPath := filepath.Join(storePath, "private.key")
	privateKeyFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer privateKeyFile.Close()

	w, err := armor.Encode(privateKeyFile, openpgp.PrivateKeyType, nil)
	if err != nil {
		return err
	}

	// Some library versions require a non-nil signer available for creating
	// self-signatures during private key serialization. If the key is encrypted,
	// temporarily decrypt using the passphrase we just collected.
	var serializeErr error
	if entity.PrivateKey != nil && entity.PrivateKey.Encrypted && s.passphrase != "" {
		if err := entity.PrivateKey.Decrypt([]byte(s.passphrase)); err == nil {
			serializeErr = entity.SerializePrivate(w, nil)
			// Re-encrypt the private key after serialization
			_ = entity.PrivateKey.Encrypt([]byte(s.passphrase))
		} else {
			serializeErr = entity.SerializePrivate(w, nil)
		}
	} else {
		serializeErr = entity.SerializePrivate(w, nil)
	}
	if serializeErr != nil {
		return serializeErr
	}
	w.Close()

	// Save public key
	publicKeyPath := filepath.Join(storePath, "public.key")
	publicKeyFile, err := os.OpenFile(publicKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer publicKeyFile.Close()

	w, err = armor.Encode(publicKeyFile, openpgp.PublicKeyType, nil)
	if err != nil {
		return err
	}

	if err := entity.Serialize(w); err != nil {
		return err
	}
	w.Close()

	return nil
}

func (s *SetupCommand) initializeSecretsFile(storePath string, entity *openpgp.Entity) error {
	secretsPath := filepath.Join(storePath, SecretsFileName)

	// Create empty secrets map
	emptySecrets := make(map[string]string)
	data, err := json.Marshal(emptySecrets)
	if err != nil {
		return err
	}

	// Encrypt it
	buf := new(bytes.Buffer)
	w, err := openpgp.Encrypt(buf, []*openpgp.Entity{entity}, nil, nil, nil)
	if err != nil {
		return err
	}

	if _, err := w.Write(data); err != nil {
		return err
	}
	w.Close()

	return os.WriteFile(secretsPath, buf.Bytes(), 0600)
}

func (s *SetupCommand) readLine() string {
	line, _ := s.reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func (s *SetupCommand) readPassword() string {
	// Hidden input for passphrase
	// Fallback to visible input if stdin is not a terminal
	if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Println()
		if err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return s.readLine()
}

func (s *SetupCommand) askYesNo(question string) bool {
	fmt.Printf("%s (y/N): ", question)
	answer := s.readLine()
	return strings.ToLower(answer) == "y" || strings.ToLower(answer) == "yes"
}

var (
	setupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Setup better-env",
		Long:  "Setup better-env to start using it. This will create a GPG key and use it to create a secure store for your environment variables.",
		RunE: func(cmd *cobra.Command, args []string) error {
			sc := NewSetupCommand()
			return sc.Run()
		},
	}
)
