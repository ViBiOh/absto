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

type Config struct {
	Directory    string
	Endpoint     string
	AccessKey    string
	SecretAccess string
	Bucket       string
	Region       string
	StorageClass string
	UseSSL       bool
	PartSize     uint64
}

func Flags(flagSet *flag.FlagSet, prefix string, overrides ...flags.Override) *Config {
	defaultFS := "/data"
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if pwd, err := os.Getwd(); err == nil {
			defaultFS = pwd
		}
	}

	var config Config

	flags.New("FileSystemDirectory", "Path to directory. Default is dynamic. `/data` on a server and Current Working Directory in a terminal.").Prefix(prefix).DocPrefix("filesystem").StringVar(flagSet, &config.Directory, defaultFS, overrides)
	flags.New("ObjectEndpoint", "Storage Object endpoint").Prefix(prefix).DocPrefix("s3").StringVar(flagSet, &config.Endpoint, "", overrides)
	flags.New("ObjectAccessKey", "Storage Object Access Key").Prefix(prefix).DocPrefix("s3").StringVar(flagSet, &config.AccessKey, "", overrides)
	flags.New("ObjectSecretAccess", "Storage Object Secret Access").Prefix(prefix).DocPrefix("s3").StringVar(flagSet, &config.SecretAccess, "", overrides)
	flags.New("ObjectBucket", "Storage Object Bucket").Prefix(prefix).DocPrefix("s3").StringVar(flagSet, &config.Bucket, "", overrides)
	flags.New("ObjectRegion", "Storage Object Region").Prefix(prefix).DocPrefix("s3").StringVar(flagSet, &config.Region, "", overrides)
	flags.New("ObjectClass", "Storage Object Class").Prefix(prefix).DocPrefix("s3").StringVar(flagSet, &config.StorageClass, "", overrides)
	flags.New("ObjectSSL", "Use SSL").Prefix(prefix).DocPrefix("s3").BoolVar(flagSet, &config.UseSSL, true, overrides)
	flags.New("PartSize", "PartSize configuration").Prefix(prefix).DocPrefix("s3").Uint64Var(flagSet, &config.PartSize, 5<<20, overrides)

	return &config
}

func New(config *Config, tracerProvider trace.TracerProvider) (storage model.Storage, err error) {
	endpoint := strings.TrimSpace(config.Endpoint)

	if len(endpoint) != 0 {
		var options []s3.ConfigOption

		if region := strings.TrimSpace(config.Region); len(region) > 0 {
			options = append(options, s3.WithRegion(region))
		}
		if storageClass := strings.TrimSpace(config.StorageClass); len(storageClass) > 0 {
			options = append(options, s3.WithStorageClass(storageClass))
		}

		storage, err = s3.New(endpoint, strings.TrimSpace(config.AccessKey), config.SecretAccess, strings.TrimSpace(config.Bucket), config.UseSSL, config.PartSize, options...)
	} else {
		storage, err = filesystem.New(strings.TrimSpace(config.Directory))
	}

	if err != nil {
		return nil, err
	}

	return telemetry.New(storage, tracerProvider), nil
}
