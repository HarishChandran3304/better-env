package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/ProtonMail/gopenpgp/v3/profile"
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
		fmt.Printf("better-env is already configured at: %s\n", storePath)
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
	fmt.Println("better-env setup")

	// Step 1: GPG Key Generation
	key, err := s.handleKeySetup()
	if err != nil {
		return fmt.Errorf("key setup failed: %w", err)
	}

	// Step 2: Save configuration
	config := Config{
		StorePath:      storePath,
		KeyFingerprint: key.GetFingerprint(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.saveConfig(storePath, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Step 3: Save the key pair
	if err := s.saveKeyPair(storePath, key); err != nil {
		return fmt.Errorf("failed to save key pair: %w", err)
	}

	// Step 4: Initialize empty secrets file
	if err := s.initializeSecretsFile(storePath, key); err != nil {
		return fmt.Errorf("failed to initialize secrets file: %w", err)
	}

	fmt.Println()
	fmt.Println("Setup complete")
	fmt.Printf("Store location: %s\n", storePath)
	fmt.Printf("Key fingerprint: %s\n", config.KeyFingerprint)
	fmt.Println()
	fmt.Println("Next steps:")
	shell := os.Getenv("SHELL")
	switch {
	// zsh
	case strings.HasSuffix(shell, "zsh"):
		fmt.Println("  Enter the following command to finish your setup:")
		fmt.Println("  echo 'bnv() { if [ \"$1\" = \"load\" ]; then eval \"$(command bnv load)\"; elif [ \"$1\" = \"unload\" ]; then eval \"$(command bnv unload)\"; else command bnv \"$@\"; fi }' >> ~/.zshrc && source ~/.zshrc")
	// bash
	case strings.HasSuffix(shell, "bash"):
		fmt.Println("  Enter the following command to finish your setup:")
		fmt.Println("  echo 'bnv() { if [ \"$1\" = \"load\" ]; then eval \"$(command bnv load)\"; elif [ \"$1\" = \"unload\" ]; then eval \"$(command bnv unload)\"; else command bnv \"$@\"; fi }' >> ~/.bashrc && source ~/.bashrc")
	// fish
	case strings.HasSuffix(shell, "fish"):
		fmt.Println("  Enter the following command to finish your setup:")
		fmt.Println("  echo 'bnv() { if [ \"$1\" = \"load\" ]; then eval \"$(command bnv load)\"; elif [ \"$1\" = \"unload\" ]; then eval \"$(command bnv unload)\"; else command bnv \"$@\"; fi }' >> ~/.config/fish/config.fish && source ~/.config/fish/config.fish")
	// other
	default:
		fmt.Println("  Add the following to your shell's config file:")
		fmt.Println(`  bnv() { if [ "$1" = "load" ]; then eval "$(command bnv load)"; elif [ "$1" = "unload" ]; then eval "$(command bnv unload)"; else command bnv "$@"; fi }`)
		return nil
	}
	fmt.Println("To learn how to use better-env check out the docs at https://better-env.dev/docs")

	return nil
}

// getStorePath returns the fixed store path for better-env
func getStorePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "better-env"), nil
}

func (s *SetupCommand) handleKeySetup() (*crypto.Key, error) {
	fmt.Println()
	fmt.Println("Generating a new GPG key...")
	return s.generateNewKey()
}

func (s *SetupCommand) generateNewKey() (*crypto.Key, error) {
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

	// Use GopenPGP's key generation with default profile (Curve25519)
	pgp := crypto.PGPWithProfile(profile.Default())
	keyGenHandle := pgp.KeyGeneration().
		AddUserId(name, email).
		New()

	key, err := keyGenHandle.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Lock the key with passphrase if provided
	if passphrase != "" {
		key, err = pgp.LockKey(key, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
	}

	return key, nil
}

func (s *SetupCommand) saveConfig(storePath string, config Config) error {
	configPath := filepath.Join(storePath, ConfigFileName)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}

func (s *SetupCommand) saveKeyPair(storePath string, key *crypto.Key) error {
	// Save private key
	privateKeyPath := filepath.Join(storePath, "private.key")
	armoredPrivate, err := key.Armor()
	if err != nil {
		return err
	}
	if err := os.WriteFile(privateKeyPath, []byte(armoredPrivate), 0600); err != nil {
		return err
	}

	// Save public key
	publicKeyPath := filepath.Join(storePath, "public.key")
	publicKey, err := key.ToPublic()
	if err != nil {
		return err
	}
	armoredPublic, err := publicKey.Armor()
	if err != nil {
		return err
	}
	return os.WriteFile(publicKeyPath, []byte(armoredPublic), 0644)
}

func (s *SetupCommand) initializeSecretsFile(storePath string, key *crypto.Key) error {
	secretsPath := filepath.Join(storePath, SecretsFileName)

	// Create empty secrets map
	emptySecrets := make(map[string]string)
	data, err := json.Marshal(emptySecrets)
	if err != nil {
		return err
	}

	// Get public key for encryption
	publicKey, err := key.ToPublic()
	if err != nil {
		return err
	}

	// Encrypt using GopenPGP
	pgp := crypto.PGP()
	encHandle, err := pgp.Encryption().Recipient(publicKey).New()
	if err != nil {
		return err
	}

	pgpMessage, err := encHandle.Encrypt(data)
	if err != nil {
		return err
	}

	return os.WriteFile(secretsPath, pgpMessage.Bytes(), 0600)
}

func (s *SetupCommand) readLine() string {
	line, _ := s.reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func (s *SetupCommand) readPassword() string {
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

func init() {
	rootCmd.AddCommand(setupCmd)
}
