package internal

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"golang.org/x/term"
)

// readPasswordFromStdin reads a password securely when attached to a TTY,
// and falls back to a normal line read otherwise. A trailing newline is trimmed.
func readPasswordFromStdin() ([]byte, error) {
	if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Println()
		return b, err
	}
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	return []byte(strings.TrimSpace(line)), err
}

// PromptAndUnlockPrivateKey prompts the user for a passphrase and attempts
// to unlock the provided armored private key up to maxAttempts times.
// Prompts are written to stderr when promptToStderr is true, otherwise stdout.
// On repeated failure, returns a user-friendly error without leaking library internals.
func PromptAndUnlockPrivateKey(privateKeyArmored []byte, promptToStderr bool, maxAttempts int) (*crypto.Key, error) {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var writer *os.File
	if promptToStderr {
		writer = os.Stderr
	} else {
		writer = os.Stdout
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Fprint(writer, "Enter passphrase: ")
		passphrase, err := readPasswordFromStdin()
		if err != nil {
			return nil, fmt.Errorf("failed to read passphrase: %w", err)
		}

		key, err := crypto.NewPrivateKeyFromArmored(string(privateKeyArmored), passphrase)
		if err == nil {
			return key, nil
		}
		if attempt < maxAttempts {
			fmt.Fprintln(writer, "Incorrect passphrase. Try again.")
		}
	}

	return nil, errors.New("incorrect passphrase")
}
