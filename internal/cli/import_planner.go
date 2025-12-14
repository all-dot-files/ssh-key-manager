package cli

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/all-dot-files/ssh-key-manager/internal/config"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

// ImportPlan holds discovered keys, host bindings, and warnings.
type ImportPlan struct {
	SourceDir string
	Keys      []ImportedKey
	Bindings  []HostBinding
	DryRun    bool
	Warnings  []string
}

// ImportedKey represents a discovered key pair for import.
type ImportedKey struct {
	Alias         string
	PrivatePath   string
	PublicPath    string
	Fingerprint   string
	Status        string // "new" or "conflict"
	DerivedPublic bool
}

// HostBinding represents a mapping from ssh config to an imported key.
type HostBinding struct {
	HostPattern string
	User        string
	Port        int
	Identity    string
	TargetAlias string
	Conflict    bool
	Action      string // keep|rebind|skip
}

// ImportPlanner orchestrates discovery and application of imports.
type ImportPlanner struct {
	manager    *config.Manager
	sourceDir  string
	aliasPref  string
	keyMap     map[string]ImportedKey // normalized path -> key
	hostConfig string
	dryRun     bool
	warnings   []string
	Keys       []ImportedKey
	Bindings   []HostBinding
}

func NewImportPlanner(manager *config.Manager, sourceDir, aliasPrefix string, dryRun bool) *ImportPlanner {
	return &ImportPlanner{
		manager:    manager,
		sourceDir:  sourceDir,
		aliasPref:  aliasPrefix,
		keyMap:     make(map[string]ImportedKey),
		hostConfig: filepath.Join(sourceDir, "config"),
		dryRun:     dryRun,
	}
}

// DiscoverKeys scans sourceDir for private/public pairs and populates plan.Keys.
func (p *ImportPlanner) DiscoverKeys() ([]ImportedKey, error) {
	entries, err := os.ReadDir(p.sourceDir)
	if err != nil {
		return nil, fmt.Errorf("read source dir: %w", err)
	}

	var keys []ImportedKey
	existing := make(map[string]struct{})
	if p.manager != nil {
		if listed, err := p.manager.ListKeys(); err == nil {
			for _, k := range listed {
				existing[k.Name] = struct{}{}
			}
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".pub") {
			continue
		}
		privPath := filepath.Join(p.sourceDir, name)
		pubPath := privPath + ".pub"
		if _, err := os.Stat(pubPath); err != nil {
			p.warnings = append(p.warnings, fmt.Sprintf("missing public key for %s", name))
			continue
		}
		if err := readable(privPath); err != nil {
			p.warnings = append(p.warnings, fmt.Sprintf("key unreadable %s: %v", privPath, err))
			continue
		}
		alias := p.aliasPref + strings.TrimSuffix(name, filepath.Ext(name))
		if _, exists := existing[alias]; exists {
			keys = append(keys, ImportedKey{
				Alias:       alias,
				PrivatePath: privPath,
				PublicPath:  pubPath,
				Fingerprint: fingerprint(pubPath),
				Status:      "conflict",
			})
			continue
		}
		keys = append(keys, ImportedKey{
			Alias:       alias,
			PrivatePath: privPath,
			PublicPath:  pubPath,
			Fingerprint: fingerprint(pubPath),
			Status:      "new",
		})
	}
	p.Keys = keys
	for _, k := range keys {
		p.keyMap[normalizePath(k.PrivatePath)] = k
		p.keyMap[normalizePath(k.PublicPath)] = k
	}
	return keys, nil
}

// DetectBindings parses ssh config and proposes bindings to imported keys.
func (p *ImportPlanner) DetectBindings() ([]HostBinding, error) {
	file, err := os.Open(p.hostConfig)
	if err != nil {
		return nil, fmt.Errorf("open ssh config: %w", err)
	}
	defer file.Close()

	var bindings []HostBinding
	currentHosts := []string{}
	var currentUser string
	var currentPort int
	var currentIdentity string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.ToLower(fields[0])
		val := strings.Join(fields[1:], " ")
		switch key {
		case "host":
			// flush previous host entries
			currentHosts = strings.Fields(val)
			currentUser = ""
			currentPort = 0
			currentIdentity = ""
		case "user":
			currentUser = val
		case "port":
			fmt.Sscanf(val, "%d", &currentPort)
		case "identityfile":
			currentIdentity = expandPath(val, p.sourceDir)
			for _, h := range currentHosts {
				k, ok := p.keyMap[normalizePath(currentIdentity)]
				if !ok {
					p.warnings = append(p.warnings, fmt.Sprintf("no imported key matches IdentityFile %s for host %s", currentIdentity, h))
					continue
				}
				bindings = append(bindings, HostBinding{
					HostPattern: h,
					User:        currentUser,
					Port:        currentPort,
					Identity:    currentIdentity,
					TargetAlias: k.Alias,
					Action:      "rebind",
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan ssh config: %w", err)
	}
	p.Bindings = bindings
	return bindings, nil
}

// Apply executes the import plan against the config manager.
func (p *ImportPlanner) Apply() error {
	if p.manager == nil {
		return fmt.Errorf("config manager not initialized")
	}
	// import keys
	for _, k := range p.Keys {
		if k.Status == "conflict" {
			continue
		}
		model := models.Key{
			Name:        k.Alias,
			Path:        k.PrivatePath,
			PubPath:     k.PublicPath,
			Fingerprint: k.Fingerprint,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := p.manager.AddKey(model); err != nil {
			return fmt.Errorf("add key %s: %w", k.Alias, err)
		}
	}
	// apply bindings
	if len(p.Bindings) > 0 {
		existingHosts, _ := p.manager.ListHosts()
		existing := make(map[string]models.Host)
		for _, h := range existingHosts {
			existing[h.Host] = h
		}
		for _, b := range p.Bindings {
			if b.Conflict {
				continue
			}
			h := models.Host{
				Host:    b.HostPattern,
				User:    b.User,
				KeyName: b.TargetAlias,
				Port:    b.Port,
			}
			if prev, ok := existing[b.HostPattern]; ok {
				if prev.KeyName == b.TargetAlias {
					continue
				}
				// overwrite with new binding
				if err := p.manager.UpdateHost(b.HostPattern, h); err != nil {
					return fmt.Errorf("update host %s: %w", b.HostPattern, err)
				}
			} else {
				if err := p.manager.AddHost(h); err != nil {
					return fmt.Errorf("add host %s: %w", b.HostPattern, err)
				}
			}
		}
	}
	return nil
}

func (p *ImportPlanner) Warnings() []string {
	return p.warnings
}

// KeyMapForTest allows tests to override key map.
func (p *ImportPlanner) KeyMapForTest(m map[string]ImportedKey) {
	p.keyMap = m
}

// Helpers

func readable(path string) error {
	_, err := os.Stat(path)
	return err
}

func normalizePath(path string) string {
	resolved := expandPath(path, "")
	return filepath.Clean(resolved)
}

func expandPath(path string, base string) string {
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	if base != "" && !strings.HasPrefix(path, "/") {
		path = filepath.Join(base, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func fingerprint(pubPath string) string {
	data, err := os.ReadFile(pubPath)
	if err != nil {
		return ""
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		return ""
	}
	return ssh.FingerprintSHA256(key)
}

// generateTestPub returns a minimal valid public key string (for tests)
func generateTestPub() string {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return ""
	}
	key, err := ssh.NewPublicKey(pub)
	if err != nil {
		return ""
	}
	return string(ssh.MarshalAuthorizedKey(key))
}

// GenerateTestPubForTest exposes a valid public key string for tests.
func GenerateTestPubForTest() string {
	return generateTestPub()
}
