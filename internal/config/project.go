package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/viper"
)

// ProjectConfig represents the application details loaded from a YAML or JSON file.
type ProjectConfig struct {
	ProjectID     string `json:"projectId" yaml:"projectId" mapstructure:"projectId"`
	ProjectName   string `json:"projectName" yaml:"projectName" mapstructure:"projectName"`
	AndroidPath   string `json:"androidPath" yaml:"androidPath" mapstructure:"androidPath"`
	IOSPath       string `json:"iosPath" yaml:"iosPath" mapstructure:"iosPath"`
	DefaultBranch string `json:"defaultBranch" yaml:"defaultBranch" mapstructure:"defaultBranch"`
}

// GetProjectID returns the resolved project/app ID.
func (c *ProjectConfig) GetProjectID() string {
	return strings.TrimSpace(c.ProjectID)
}

// GetProjectName returns the project display name.
func (c *ProjectConfig) GetProjectName() string {
	return strings.TrimSpace(c.ProjectName)
}

// GetAndroidPath returns the configured android build path.
func (c *ProjectConfig) GetAndroidPath() string {
	return strings.TrimSpace(c.AndroidPath)
}

// GetIOSPath returns the configured iOS build path.
func (c *ProjectConfig) GetIOSPath() string {
	return strings.TrimSpace(c.IOSPath)
}

// GetDefaultBranch returns the default branch if configured.
func (c *ProjectConfig) GetDefaultBranch() string {
	return strings.TrimSpace(c.DefaultBranch)
}

// CandidateConfigFileNames defines standard configuration filenames.
var CandidateConfigFileNames = []string{
	"buildshare.json",
	"buildshare.yaml",
	"buildshare.yml",
	".buildshare.json",
	".buildshare.yaml",
	".buildshare.yml",
	"app.json",
	"app.yaml",
	"app.yml",
}

// FindProjectConfigFile searches for an existing project configuration file.
func FindProjectConfigFile(customPath string) (string, error) {
	if customPath != "" {
		expanded := expandHome(customPath)
		if fi, err := os.Stat(expanded); err == nil && !fi.IsDir() {
			return expanded, nil
		}
		return "", fmt.Errorf("configuration file not found: %s", customPath)
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	dir := cwd
	for i := 0; i < 5; i++ {
		for _, name := range CandidateConfigFileNames {
			p := filepath.Join(dir, name)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				return p, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no project configuration file found (checked buildshare.json, buildshare.yaml, .buildshare.json, etc.). Use --config to specify a file")
}

// LoadProjectConfig loads and parses project configuration from a JSON or YAML file.
func LoadProjectConfig(customPath string) (*ProjectConfig, string, error) {
	configPath, err := FindProjectConfigFile(customPath)
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, configPath, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var cfg ProjectConfig
	// Attempt JSON unmarshal first
	jsonErr := json.Unmarshal(data, &cfg)
	if jsonErr == nil && (cfg.ProjectID != "" || cfg.ProjectName != "" || cfg.AndroidPath != "" || cfg.IOSPath != "") {
		return &cfg, configPath, nil
	}

	// Use Viper for YAML or alternate format parsing
	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		if jsonErr != nil {
			return nil, configPath, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}
	}

	_ = v.Unmarshal(&cfg)

	// Field resolution using camelCase keys
	if cfg.ProjectID == "" {
		cfg.ProjectID = v.GetString("projectId")
	}
	if cfg.ProjectName == "" {
		cfg.ProjectName = v.GetString("projectName")
	}
	if cfg.AndroidPath == "" {
		cfg.AndroidPath = v.GetString("androidPath")
	}
	if cfg.IOSPath == "" {
		cfg.IOSPath = v.GetString("iosPath")
	}
	if cfg.DefaultBranch == "" {
		cfg.DefaultBranch = v.GetString("defaultBranch")
	}

	return &cfg, configPath, nil
}

// ResolveBuildArtifact locates the APK or IPA file for a given platform and configured path.
// If configuredPath points to a file, it verifies its existence and returns it.
// If configuredPath points to a directory, it searches for matching build artifacts (.apk or .ipa)
// and returns the most recent release candidate.
func ResolveBuildArtifact(platform, configuredPath, configDir string) (string, error) {
	if configuredPath == "" {
		return "", fmt.Errorf("%s path is not configured", platform)
	}

	resolvedPath := expandHome(configuredPath)
	if !filepath.IsAbs(resolvedPath) && configDir != "" {
		candidate := filepath.Join(configDir, resolvedPath)
		if _, err := os.Stat(candidate); err == nil {
			resolvedPath = candidate
		}
	}

	fi, err := os.Stat(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("configured %s path does not exist: %s", platform, resolvedPath)
	}

	normPlatform := strings.ToLower(strings.TrimSpace(platform))

	// If configuredPath is directly a file, return it
	if !fi.IsDir() {
		return resolvedPath, nil
	}

	var validExts []string
	switch normPlatform {
	case "android":
		validExts = []string{".apk", ".aab"}
	case "ios":
		validExts = []string{".ipa"}
	default:
		validExts = []string{".apk", ".ipa", ".aab"}
	}

	type fileCandidate struct {
		path    string
		modTime int64
		score   int
	}

	var candidates []fileCandidate
	baseDepth := strings.Count(filepath.Clean(resolvedPath), string(filepath.Separator))

	err = filepath.WalkDir(resolvedPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		currentDepth := strings.Count(filepath.Clean(path), string(filepath.Separator))
		if d.IsDir() {
			if currentDepth-baseDepth > 3 {
				return filepath.SkipDir
			}
			return nil
		}

		info, statErr := d.Info()
		if statErr != nil || info.Size() == 0 {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		matched := false
		for _, v := range validExts {
			if ext == v {
				matched = true
				break
			}
		}
		if !matched {
			return nil
		}

		nameLower := strings.ToLower(d.Name())
		score := 100
		if ext == ".apk" || ext == ".ipa" {
			score += 50
		}
		if strings.Contains(nameLower, "release") {
			score += 30
		}
		if strings.Contains(nameLower, "unsigned") || strings.Contains(nameLower, "unaligned") {
			score -= 50
		}

		candidates = append(candidates, fileCandidate{
			path:    path,
			modTime: info.ModTime().UnixNano(),
			score:   score,
		})
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed searching %s for builds: %w", resolvedPath, err)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no build artifact (%s) found in %s", strings.Join(validExts, ", "), resolvedPath)
	}

	// Sort candidates: highest score first, then newest modTime first
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].modTime > candidates[j].modTime
	})

	return candidates[0].path, nil
}

func expandHome(path string) string {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// DetectTargetConfigFile returns the preferred file to write project configuration.
// If an existing configuration file exists, it returns that path.
// Otherwise, it defaults to "buildshare.json".
func DetectTargetConfigFile() string {
	for _, name := range CandidateConfigFileNames {
		if fi, err := os.Stat(name); err == nil && !fi.IsDir() {
			return name
		}
	}
	return "buildshare.json"
}

// SaveProjectConfig writes the project configuration to a file (JSON or YAML based on file extension).
func SaveProjectConfig(cfg *ProjectConfig, filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".yaml" || ext == ".yml" {
		content := fmt.Sprintf("# BuildShare project configuration\nprojectId: %q\nprojectName: %q\nandroidPath: %q\niosPath: %q\ndefaultBranch: %q\n",
			cfg.GetProjectID(),
			cfg.GetProjectName(),
			cfg.GetAndroidPath(),
			cfg.GetIOSPath(),
			cfg.GetDefaultBranch(),
		)
		return os.WriteFile(filePath, []byte(content), 0644)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filePath, data, 0644)
}

