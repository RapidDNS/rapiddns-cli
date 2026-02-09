package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	APIKey = "api_key"
)

// InitConfig initializes the configuration
func InitConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	viper.AddConfigPath(home)
	viper.SetConfigType("yaml")
	viper.SetConfigName(".rapiddns")

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// SetAPIKey sets the API key in the configuration file
func SetAPIKey(key string) error {
	viper.Set(APIKey, key)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".rapiddns.yaml")
	return viper.WriteConfigAs(configPath)
}

// GetAPIKey returns the API key from the configuration
func GetAPIKey() string {
	return viper.GetString(APIKey)
}
