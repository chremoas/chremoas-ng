package config

import (
	"context"
	"fmt"
	"os"

	sl "github.com/bhechinger/spiffylogger"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	// Import the remote config driver
	_ "github.com/spf13/viper/remote"
)

func New(ctx context.Context, filename string) error {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(zap.String("filename", filename))

	var fileRead, remoteRead bool
	var fileReadErr, remoteReadErr error

	configNameSpace := os.Getenv("CONFIG_NAMESPACE")
	if configNameSpace == "" {
		configNameSpace = "default"
	}

	configType := os.Getenv("CONFIG_TYPE")
	if configType == "" {
		configType = "yaml"
	}

	sp.With(zap.String("config_type", configType))

	viper.SetConfigFile(filename)

	if fileReadErr = viper.ReadInConfig(); fileReadErr == nil {
		sp.Info("Successfully read local config file")
		fileRead = true
	}

	if err := viper.BindEnv("CONSUL"); err == nil {
		consul := viper.Get("consul")
		sp.With(zap.Any("consul", consul))

		if consul != nil {
			// TODO: This is very rigid. Let's find a better way.
			configPath := fmt.Sprintf("/%s/config", configNameSpace)
			sp.With(zap.String("config_path", configPath))
			sp.Info("Using config")
			err := viper.AddRemoteProvider("consul", consul.(string), configPath)
			if err != nil {
				sp.Info(err.Error())
			} else {
				viper.SetConfigType(configType) // because there is no file extension in a stream of bytes, supported extensions are "json", "toml", "yaml", "yml", "properties", "props", "prop"

				if remoteReadErr = viper.ReadRemoteConfig(); remoteReadErr == nil {
					sp.Info("Successfully read remote config")
					remoteRead = true
				}
			}
		}
	}

	if !fileRead && !remoteRead {
		sp.Error(
			"unable to read config",
			zap.NamedError("file_read_error", fileReadErr),
			zap.NamedError("remote_read_error", remoteReadErr),
		)
		return fmt.Errorf("unable to read config:\n\tfile=%v\n\tremote=%v", fileReadErr, remoteReadErr)
	}

	return nil
}
