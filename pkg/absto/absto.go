package absto

import (
	"flag"
	"os"
	"strings"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/absto/pkg/s3"
	"github.com/ViBiOh/absto/pkg/telemetry"
	"github.com/ViBiOh/flags"
	"go.opentelemetry.io/otel/trace"
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
	partSize     *uint64
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
		directory: flags.String(fs, prefix, "filesystem", "FileSystemDirectory", "Path to directory. Default is dynamic. `/data` on a server and Current Working Directory in a terminal.", defaultFS, overrides),

		endpoint:     flags.String(fs, prefix, "s3", "ObjectEndpoint", "Storage Object endpoint", "", overrides),
		accessKey:    flags.String(fs, prefix, "s3", "ObjectAccessKey", "Storage Object Access Key", "", overrides),
		secretAccess: flags.String(fs, prefix, "s3", "ObjectSecretAccess", "Storage Object Secret Access", "", overrides),
		bucket:       flags.String(fs, prefix, "s3", "ObjectBucket", "Storage Object Bucket", "", overrides),
		useSSL:       flags.Bool(fs, prefix, "s3", "ObjectSSL", "Use SSL", true, overrides),
		partSize:     flags.Uint64(fs, prefix, "s3", "PartSize", "PartSize configuration", 5<<20, overrides),
	}
}

// New creates new Storage from Config
func New(config Config, tracer trace.Tracer) (storage model.Storage, err error) {
	endpoint := strings.TrimSpace(*config.endpoint)
	if len(endpoint) != 0 {
		storage, err = s3.New(endpoint, strings.TrimSpace(*config.accessKey), *config.secretAccess, strings.TrimSpace(*config.bucket), *config.useSSL, *config.partSize)
	} else {
		storage, err = filesystem.New(strings.TrimSpace(*config.directory))
	}

	if err != nil {
		return
	}

	storage = telemetry.New(storage, tracer)

	return
}
