package xdg

import (
	"os"
	"path/filepath"
)

// XDGDirs provides access to XDG Base Directory Specification compliant paths
type XDGDirs struct {
	dataHome   string
	configHome string
	stateHome  string
	cacheHome  string
	runtimeDir string
	dataDirs   []string
	configDirs []string
}

// NewXDGDirs creates a new XDGDirs instance with proper defaults according to XDG spec
func NewXDGDirs() *XDGDirs {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current user's home from environment
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/tmp" // Last resort fallback
		}
	}

	xdg := &XDGDirs{}

	// XDG_DATA_HOME: user-specific data files
	xdg.dataHome = os.Getenv("XDG_DATA_HOME")
	if xdg.dataHome == "" {
		xdg.dataHome = filepath.Join(homeDir, ".local", "share")
	}

	// XDG_CONFIG_HOME: user-specific configuration files
	xdg.configHome = os.Getenv("XDG_CONFIG_HOME")
	if xdg.configHome == "" {
		xdg.configHome = filepath.Join(homeDir, ".config")
	}

	// XDG_STATE_HOME: user-specific state data
	xdg.stateHome = os.Getenv("XDG_STATE_HOME")
	if xdg.stateHome == "" {
		xdg.stateHome = filepath.Join(homeDir, ".local", "state")
	}

	// XDG_CACHE_HOME: user-specific non-essential (cached) data
	xdg.cacheHome = os.Getenv("XDG_CACHE_HOME")
	if xdg.cacheHome == "" {
		xdg.cacheHome = filepath.Join(homeDir, ".cache")
	}

	// XDG_RUNTIME_DIR: user-specific runtime files and other file objects
	xdg.runtimeDir = os.Getenv("XDG_RUNTIME_DIR")
	if xdg.runtimeDir == "" {
		// Create a fallback runtime directory if not set
		// According to spec, we should print a warning and use a replacement
		xdg.runtimeDir = filepath.Join("/tmp", "tester-runtime-"+os.Getenv("USER"))
	}

	// XDG_DATA_DIRS: preference-ordered base directories to search for data files
	dataDirsEnv := os.Getenv("XDG_DATA_DIRS")
	if dataDirsEnv == "" {
		xdg.dataDirs = []string{"/usr/local/share", "/usr/share"}
	} else {
		xdg.dataDirs = filepath.SplitList(dataDirsEnv)
	}

	// XDG_CONFIG_DIRS: preference-ordered base directories to search for configuration files
	configDirsEnv := os.Getenv("XDG_CONFIG_DIRS")
	if configDirsEnv == "" {
		xdg.configDirs = []string{"/etc/xdg"}
	} else {
		xdg.configDirs = filepath.SplitList(configDirsEnv)
	}

	return xdg
}

// DataHome returns the base directory for user-specific data files
func (x *XDGDirs) DataHome() string {
	return x.dataHome
}

// ConfigHome returns the base directory for user-specific configuration files
func (x *XDGDirs) ConfigHome() string {
	return x.configHome
}

// StateHome returns the base directory for user-specific state files
func (x *XDGDirs) StateHome() string {
	return x.stateHome
}

// CacheHome returns the base directory for user-specific cached data
func (x *XDGDirs) CacheHome() string {
	return x.cacheHome
}

// RuntimeDir returns the base directory for user-specific runtime files
func (x *XDGDirs) RuntimeDir() string {
	return x.runtimeDir
}

// DataDirs returns the preference-ordered base directories for data files
func (x *XDGDirs) DataDirs() []string {
	return append([]string{x.dataHome}, x.dataDirs...)
}

// ConfigDirs returns the preference-ordered base directories for configuration files
func (x *XDGDirs) ConfigDirs() []string {
	return append([]string{x.configHome}, x.configDirs...)
}

// AppDataDir returns the application-specific data directory
func (x *XDGDirs) AppDataDir(appName string) string {
	return filepath.Join(x.dataHome, appName)
}

// AppConfigDir returns the application-specific config directory
func (x *XDGDirs) AppConfigDir(appName string) string {
	return filepath.Join(x.configHome, appName)
}

// AppStateDir returns the application-specific state directory
func (x *XDGDirs) AppStateDir(appName string) string {
	return filepath.Join(x.stateHome, appName)
}

// AppCacheDir returns the application-specific cache directory
func (x *XDGDirs) AppCacheDir(appName string) string {
	return filepath.Join(x.cacheHome, appName)
}

// AppRuntimeDir returns the application-specific runtime directory
func (x *XDGDirs) AppRuntimeDir(appName string) string {
	return filepath.Join(x.runtimeDir, appName)
}

// EnsureDir creates the directory with appropriate permissions if it doesn't exist
func (x *XDGDirs) EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// EnsureRuntimeDir creates the runtime directory with secure permissions (0700)
func (x *XDGDirs) EnsureRuntimeDir(path string) error {
	return os.MkdirAll(path, 0700)
}
