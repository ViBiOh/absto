package absto

import (
	"flag"
	"os"
	"strings"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/absto/pkg/s3"
	"github.com/ViBiOh/flags"
	"golang.org/x/term"
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
	defaultFS := "/data"
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if pwd, err := os.Getwd(); err == nil {
			defaultFS = pwd
		}
	}

	return Config{
		directory: flags.New(prefix, "filesystem", "FileSystemDirectory").Default(defaultFS, overrides).Label("Path to directory. Default is dynamic. `/data` on a server and Current Working Directory in a terminal.").ToString(fs),

		endpoint:     flags.New(prefix, "s3", "ObjectEndpoint").Default("", overrides).Label("Storage Object endpoint").ToString(fs),
		accessKey:    flags.New(prefix, "s3", "ObjectAccessKey").Default("", overrides).Label("Storage Object Access Key").ToString(fs),
		secretAccess: flags.New(prefix, "s3", "ObjectSecretAccess").Default("", overrides).Label("Storage Object Secret Access").ToString(fs),
		bucket:       flags.New(prefix, "s3", "ObjectBucket").Default("", overrides).Label("Storage Object Bucket").ToString(fs),
		useSSL:       flags.New(prefix, "s3", "ObjectSSL").Default(true, overrides).Label("Use SSL").ToBool(fs),
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
