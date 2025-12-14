package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage project-level configuration",
	Long: `Manage project-level SKM configuration (.skmconfig).

Project configuration allows you to:
  - Override global settings per project
  - Share SSH configuration with team members
  - Define project-specific hosts and keys
  - Set project-specific Git user/email`,
}

var projectInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project configuration",
	Long:  `Create a .skmconfig file in the current directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, _ := cmd.Flags().GetString("name")
		defaultKey, _ := cmd.Flags().GetString("default-key")
		gitUser, _ := cmd.Flags().GetString("git-user")
		gitEmail, _ := cmd.Flags().GetString("git-email")
		teamName, _ := cmd.Flags().GetString("team")

		projectConfig := &models.ProjectConfig{
			ProjectName: projectName,
			DefaultKey:  defaultKey,
			GitUser:     gitUser,
			GitEmail:    gitEmail,
			TeamName:    teamName,
		}

		if err := configManager.CreateProjectConfig("", projectConfig); err != nil {
			return err
		}

		cwd, _ := os.Getwd()
		fmt.Printf("✓ Created project configuration: %s/.skmconfig\n", cwd)

		if projectName != "" {
			fmt.Printf("  Project: %s\n", projectName)
		}
		if defaultKey != "" {
			fmt.Printf("  Default Key: %s\n", defaultKey)
		}
		if teamName != "" {
			fmt.Printf("  Team: %s\n", teamName)
		}

		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit .skmconfig to customize settings")
		fmt.Println("  2. Add .skmconfig to version control for team sharing")
		fmt.Println("  3. Add .skmconfig to .gitignore if it contains sensitive info")

		return nil
	},
}

var projectShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current project configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !configManager.HasProjectConfig() {
			fmt.Println("No project configuration found in current or parent directories.")
			fmt.Println("Run 'skm project init' to create one.")
			return nil
		}

		projectConfig := configManager.GetProjectConfig()
		projectPath := configManager.GetProjectPath()

		fmt.Printf("📁 Project Configuration\n")
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("Location: %s/.skmconfig\n\n", projectPath)

		if projectConfig.ProjectName != "" {
			fmt.Printf("Project Name:   %s\n", projectConfig.ProjectName)
		}
		if projectConfig.TeamName != "" {
			fmt.Printf("Team:           %s\n", projectConfig.TeamName)
		}
		if projectConfig.DefaultKey != "" {
			fmt.Printf("Default Key:    %s\n", projectConfig.DefaultKey)
		}
		if projectConfig.User != "" {
			fmt.Printf("User Override:  %s\n", projectConfig.User)
		}
		if projectConfig.Email != "" {
			fmt.Printf("Email Override: %s\n", projectConfig.Email)
		}
		if projectConfig.GitUser != "" {
			fmt.Printf("Git User:       %s\n", projectConfig.GitUser)
		}
		if projectConfig.GitEmail != "" {
			fmt.Printf("Git Email:      %s\n", projectConfig.GitEmail)
		}
		if projectConfig.DefaultKeyPolicy != "" {
			fmt.Printf("Key Policy:     %s\n", projectConfig.DefaultKeyPolicy)
		}

		if len(projectConfig.Hosts) > 0 {
			fmt.Printf("\nProject Hosts:  %d configured\n", len(projectConfig.Hosts))
		}

		return nil
	},
}

var projectValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate project configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !configManager.HasProjectConfig() {
			return fmt.Errorf("no project configuration found")
		}

		projectConfig := configManager.GetProjectConfig()
		projectPath := configManager.GetProjectPath()

		fmt.Printf("Validating project configuration at: %s\n\n", projectPath)

		warnings := 0
		errors := 0

		// Check for valid key policy
		if projectConfig.DefaultKeyPolicy != "" {
			validPolicies := []models.KeyPolicy{models.KeyPolicyAuto, models.KeyPolicyAsk, models.KeyPolicyNever}
			valid := false
			for _, p := range validPolicies {
				if projectConfig.DefaultKeyPolicy == p {
					valid = true
					break
				}
			}
			if !valid {
				fmt.Printf("✗ Invalid default_key_policy: %s\n", projectConfig.DefaultKeyPolicy)
				errors++
			}
		}

		// Check if default key exists
		if projectConfig.DefaultKey != "" {
			if _, err := configManager.GetKey(projectConfig.DefaultKey); err != nil {
				fmt.Printf("⚠️  Default key '%s' not found in global config\n", projectConfig.DefaultKey)
				warnings++
			}
		}

		// Validate hosts
		for i, host := range projectConfig.Hosts {
			if host.Host == "" {
				fmt.Printf("✗ Host #%d: missing 'host' field\n", i+1)
				errors++
			}
			if host.KeyName != "" {
				if _, err := configManager.GetKey(host.KeyName); err != nil {
					fmt.Printf("⚠️  Host '%s': key '%s' not found\n", host.Host, host.KeyName)
					warnings++
				}
			}
		}

		// Summary
		fmt.Println()
		if errors == 0 && warnings == 0 {
			fmt.Println("✓ Configuration is valid")
		} else {
			if errors > 0 {
				fmt.Printf("✗ Found %d error(s)\n", errors)
			}
			if warnings > 0 {
				fmt.Printf("⚠️  Found %d warning(s)\n", warnings)
			}
		}

		if errors > 0 {
			return fmt.Errorf("validation failed")
		}

		return nil
	},
}

var projectExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Show example project configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		example := models.ProjectConfig{
			ProjectName:      "my-project",
			TeamName:         "dev-team",
			DefaultKey:       "work",
			GitUser:          "John Doe",
			GitEmail:         "john@company.com",
			DefaultKeyPolicy: models.KeyPolicyAuto,
			AutoCreateHost:   true,
			KeyType:          models.KeyTypeED25519,
			Hosts: []models.Host{
				{
					Host:    "prod-server",
					User:    "deploy",
					KeyName: "work",
					Port:    22,
				},
				{
					Host:    "staging-server",
					User:    "deploy",
					KeyName: "work",
					Port:    22,
				},
			},
		}

		data, err := yaml.Marshal(&example)
		if err != nil {
			return err
		}

		fmt.Println("# Example .skmconfig file")
		fmt.Println("#")
		fmt.Println("# This file can be committed to version control to share")
		fmt.Println("# SSH configuration with your team.")
		fmt.Println()
		fmt.Println(string(data))

		return nil
	},
}

var projectMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Show merged configuration (global + project)",
	RunE: func(cmd *cobra.Command, args []string) error {
		merged := configManager.GetMerged()

		fmt.Println("🔀 Merged Configuration (Global + Project)")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("User:        %s\n", merged.User)
		fmt.Printf("Email:       %s\n", merged.Email)
		fmt.Printf("Key Policy:  %s\n", merged.DefaultKeyPolicy)
		fmt.Printf("Debug Mode:  %v\n", merged.Debug)
		fmt.Printf("Total Keys:  %d\n", len(merged.Keys))
		fmt.Printf("Total Hosts: %d\n", len(merged.Hosts))

		if configManager.HasProjectConfig() {
			fmt.Printf("\n✓ Project config loaded from: %s\n", configManager.GetProjectPath())
		} else {
			fmt.Printf("\nℹ️  No project config found (using global config only)\n")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(projectCmd)

	// Init command
	projectCmd.AddCommand(projectInitCmd)
	projectInitCmd.Flags().String("name", "", "Project name")
	projectInitCmd.Flags().String("default-key", "", "Default key for this project")
	projectInitCmd.Flags().String("git-user", "", "Git user name for this project")
	projectInitCmd.Flags().String("git-email", "", "Git email for this project")
	projectInitCmd.Flags().String("team", "", "Team name")

	// Show command
	projectCmd.AddCommand(projectShowCmd)

	// Validate command
	projectCmd.AddCommand(projectValidateCmd)

	// Example command
	projectCmd.AddCommand(projectExampleCmd)

	// Merge command
	projectCmd.AddCommand(projectMergeCmd)
}
