package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
	"github.com/all-dot-files/ssh-key-manager/internal/sshconfig"
)

// updateSSHConfig syncs the current SKM configuration to the ~/.ssh/config file
func updateSSHConfig() error {
	cfg := configManager.Get()
	sshMgr := sshconfig.NewManager(cfg.SSHDir)

	keyMap := make(map[string]*models.Key)
	for i := range cfg.Keys {
		keyMap[cfg.Keys[i].Name] = &cfg.Keys[i]
	}

	return sshMgr.UpdateConfig(cfg.Hosts, keyMap)
}

// promptUser asks the user for input with an optional default value
func promptUser(prompt string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}
