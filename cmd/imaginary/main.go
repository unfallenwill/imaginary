package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	d "runtime/debug"
	"strings"
	"time"

	"github.com/h2non/imaginary/internal/config"
	img "github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/server"
	"github.com/h2non/imaginary/internal/source"
	bodysource "github.com/h2non/imaginary/internal/source/body"
	fssource "github.com/h2non/imaginary/internal/source/fs"
	httpsource "github.com/h2non/imaginary/internal/source/http"
	objectsource "github.com/h2non/imaginary/internal/source/object"
	"github.com/h2non/imaginary/internal/storage"
	"github.com/h2non/imaginary/internal/version"
)

const usage = `imaginary %s

Usage:
  imaginary -p 80
  imaginary -cors
  imaginary -concurrency 10
  imaginary -path-prefix /api/v1
  imaginary -enable-url-source
  imaginary -disable-endpoints form,health,crop,rotate
  imaginary -enable-url-source -allowed-origins http://localhost,http://server.com
  imaginary -enable-url-source -enable-auth-forwarding
  imaginary -enable-url-source -authorization "Basic AwDJdL2DbwrD=="
  imaginary -enable-placeholder
  imaginary -enable-url-source -placeholder ./placeholder.jpg
  imaginary -enable-url-signature -url-signature-key 4f46feebafc4b5e988f131c4ff8b5997
  imaginary -enable-url-source -forward-headers X-Custom,X-Token
  imaginary -h | -help
  imaginary -v | -version

Options:

  -a <addr>                  Bind address [default: *]
  -p <port>                  Bind port [default: 8088]
  -h, -help                  Show help
  -v, -version               Show version
  -path-prefix <value>       Url path prefix to listen to [default: "/"]
  -cors                      Enable CORS support [default: false]
  -gzip                      Enable gzip compression (deprecated) [default: false]
  -disable-endpoints         Comma separated endpoints to disable. E.g: form,crop,rotate,health [default: ""]
  -key <key>                 Define API key for authorization
  -mount <path>              Mount server local directory
  -http-cache-ttl <num>      The TTL in seconds. Adds caching headers to locally served files.
  -http-read-timeout <num>   HTTP read timeout in seconds [default: 30]
  -http-write-timeout <num>  HTTP write timeout in seconds [default: 30]
  -enable-url-source         Enable remote HTTP URL image source processing
  -enable-placeholder        Enable image response placeholder to be used in case of error [default: false]
  -enable-auth-forwarding    Forwards X-Forward-Authorization or Authorization header to the image source server. -enable-url-source flag must be defined. Tip: secure your server from public access to prevent attack vectors
  -forward-headers           Forwards custom headers to the image source server. -enable-url-source flag must be defined.
  -enable-url-signature      Enable URL signature (URL-safe Base64-encoded HMAC digest) [default: false]
  -url-signature-key         The URL signature key (32 characters minimum)
  -allowed-origins <urls>    Restrict remote image source processing to certain origins (separated by commas)
  -max-allowed-size <bytes>  Restrict maximum size of http image source (in bytes)
  -max-allowed-resolution <megapixels> Restrict maximum resolution of the image [default: 18.0]
  -certfile <path>           TLS certificate file path
  -keyfile <path>            TLS private key file path
  -authorization <value>     Defines a constant Authorization header value passed to all the image source servers. -enable-url-source flag must be defined. This overwrites authorization headers forwarding behavior via X-Forward-Authorization
  -placeholder <path>        Image path to image custom placeholder to be used in case of error. Recommended minimum image size is: 1200x1200
  -placeholder-status <code> HTTP status returned when use -placeholder flag
  -concurrency <num>         Max concurrent image processing requests [default: disabled]
  -burst <num>               (deprecated, ignored) Throttle burst max cache size [default: 100]
  -mrelease <num>            OS memory release interval in seconds [default: 30]
  -cpus <num>                Number of used cpu cores.
                             (default for current machine is %d cores)
  -log-level                 Set log level for http-server. E.g: info,warning,error [default: info].
                             Or can use the environment variable GOLANG_LOG=info.
  -return-size               Return the image size with X-Width and X-Height HTTP header. [default: disabled].
  -config <path>             YAML config file path.
                             Default: $HOME/.config/imaginary/config.yaml
`

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if cfg.ShowHelp {
		fmt.Fprintf(os.Stderr, usage, version.Version, runtime.NumCPU())
		os.Exit(1)
	}
	if cfg.ShowVersion {
		fmt.Println(version.Version)
		os.Exit(0)
	}

	if err := cfg.Validate(); err != nil {
		exitWithError(err.Error())
	}

	runtime.GOMAXPROCS(cfg.CPUs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.MRelease > 0 {
		startMemoryRelease(ctx, cfg.MRelease)
	}

	placeholderImage, placeholderEnabled, err := cfg.ResolvePlaceholder()
	if err != nil {
		exitWithError(err.Error())
	}

	objectStorage, err := resolveObjectStorage(cfg)
	if err != nil {
		exitWithError(err.Error())
	}

	resolver := buildResolver(cfg, objectStorage)

	serverCfg := server.Config{
		Addr:               cfg.Addr,
		Port:               cfg.Port,
		HTTPReadTimeout:    cfg.HTTPReadTimeout,
		HTTPWriteTimeout:   cfg.HTTPWriteTimeout,
		CertFile:           cfg.CertFile,
		KeyFile:            cfg.KeyFile,
		LogLevel:           cfg.LogLevel,
		PathPrefix:         cfg.PathPrefix,
		CORS:               cfg.CORS,
		APIKey:             cfg.APIKey,
		Concurrency:        cfg.Concurrency,
		HTTPCacheTTL:       cfg.HTTPCacheTTL,
		EnableURLSource:    cfg.EnableURLSource,
		Mount:              cfg.Mount,
		EnableURLSignature: cfg.EnableURLSignature,
		URLSignatureKey:    cfg.URLSignatureKey,
		Endpoints:          server.EndpointSet(cfg.ParseEndpoints()),
		MaxAllowedPixels:   cfg.MaxAllowedPixels,
		MaxAllowedSize:     cfg.MaxAllowedSize,
		ReturnSize:         cfg.ReturnSize,
		Error: server.ErrorConfig{
			PlaceholderEnabled: placeholderEnabled,
			PlaceholderImage:   placeholderImage,
			PlaceholderStatus:  cfg.PlaceholderStatus,
			Resizer:            img.BimgResizer{},
		},
		ObjectStorage: objectStorage,
		Resolver:      resolver,
	}

	debug("imaginary server listening on port :%d/%s", serverCfg.Port, strings.TrimPrefix(serverCfg.PathPrefix, "/"))

	server.Server(serverCfg)
	cancel()
}

func resolveObjectStorage(cfg config.Config) (source.ObjectStorage, error) {
	if cfg.Storage.Type == "" {
		return nil, nil
	}
	return storage.NewProvider(context.Background(), cfg.Storage)
}

func buildResolver(cfg config.Config, objectStorage source.ObjectStorage) *source.Resolver {
	return source.NewResolver(
		bodysource.NewBodyImageSource(cfg.MaxAllowedSize),
		objectsource.NewObjectImageSource(objectStorage, cfg.MaxAllowedSize),
		fssource.NewFileSystemImageSource(cfg.Mount),
		httpsource.NewHTTPImageSource(httpsource.Config{
			AuthForwarding: cfg.AuthForwarding,
			Authorization:  cfg.Authorization,
			ForwardHeaders: cfg.ForwardHeaders,
			AllowedOrigins: cfg.AllowedOrigins,
			MaxAllowedSize: cfg.MaxAllowedSize,
		}),
	)
}

func startMemoryRelease(ctx context.Context, interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				debug("FreeOSMemory()")
				d.FreeOSMemory()
			}
		}
	}()
}

func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func debug(msg string, values ...interface{}) {
	if dbg := os.Getenv("DEBUG"); dbg == "imaginary" || dbg == "*" {
		log.Printf(msg, values...)
	}
}
