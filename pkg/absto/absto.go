package absto

import (
	"flag"
	"strings"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/absto/pkg/s3"
	"github.com/ViBiOh/flags"
)

// Config of package
type Config struct {
	directory *string

	endpoint     *string
	accessKey    *string
	secretAccess *string
	bucket       *string
	useSSL       *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		directory: flags.New(prefix, "filesystem", "Directory").Default("/data", overrides).Label("Path to directory").ToString(fs),

		endpoint:     flags.New(prefix, "s3", "Endpoint").Default("", overrides).Label("Storage Object endpoint").ToString(fs),
		accessKey:    flags.New(prefix, "s3", "AccessKey").Default("", overrides).Label("Storage Object Access Key").ToString(fs),
		secretAccess: flags.New(prefix, "s3", "SecretAccess").Default("", overrides).Label("Storage Object Secret Access").ToString(fs),
		bucket:       flags.New(prefix, "s3", "Bucket").Default("", overrides).Label("Storage Object Bucket").ToString(fs),
		useSSL:       flags.New(prefix, "s3", "SSL").Default(true, overrides).Label("Use SSL").ToBool(fs),
	}
}

// New creates new Storage from Config
func New(config Config) (model.Storage, error) {
	endpoint := strings.TrimSpace(*config.endpoint)
	if len(endpoint) != 0 {
		return s3.New(endpoint, strings.TrimSpace(*config.accessKey), *config.secretAccess, strings.TrimSpace(*config.bucket), *config.useSSL)
	}

	return filesystem.New(strings.TrimSpace(*config.directory))
}
