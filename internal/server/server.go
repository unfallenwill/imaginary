package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	img "github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
)

// Config holds the full configuration for the HTTP image server.
type Config struct {
	// Network
	Addr             string
	Port             int
	HTTPReadTimeout  int
	HTTPWriteTimeout int
	CertFile         string
	KeyFile          string
	LogLevel         string

	// Routing
	PathPrefix string

	// Middleware
	CORS               bool
	APIKey             string
	Concurrency        int
	Burst              int
	HTTPCacheTTL       int
	EnableURLSource    bool
	Mount              string
	EnableURLSignature bool
	URLSignatureKey    string
	Endpoints          EndpointSet

	// Image processing
	MaxAllowedPixels float64
	MaxAllowedSize   int
	ReturnSize       bool

	// Error handling
	Error ErrorConfig

	// Object storage
	ObjectStorage source.ObjectStorage

	// Source resolver (wired externally)
	Resolver *source.Resolver

	// HTTP client for fetching remote resources (e.g. watermark images).
	// If nil, a default client with a 15s timeout is used.
	RemoteClient *http.Client
}

func Server(cfg Config) {
	if cfg.RemoteClient == nil {
		cfg.RemoteClient = &http.Client{
			Timeout: 15 * time.Second,
		}
	}
	InitStartTime()

	addr := cfg.Addr + ":" + strconv.Itoa(cfg.Port)
	handler := NewLog(NewServerMux(cfg), os.Stdout, cfg.LogLevel)

	server := &http.Server{
		Addr:           addr,
		Handler:        handler,
		MaxHeaderBytes: 1 << 20,
		ReadTimeout:    time.Duration(cfg.HTTPReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.HTTPWriteTimeout) * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := listenAndServe(server, cfg); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-done
	log.Print("Graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
}

func listenAndServe(s *http.Server, cfg Config) error {
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		return s.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
	}
	return s.ListenAndServe()
}

func join(prefix, route string) string {
	return path.Join(prefix, route)
}

// NewServerMux creates a new HTTP server route multiplexer.
func NewServerMux(cfg Config) http.Handler {
	mux := http.NewServeMux()

	prefix := cfg.PathPrefix
	resolver := cfg.Resolver

	mux.Handle(join(prefix, "/"), Middleware(indexController(prefix, cfg.Error), cfg))
	mux.Handle(join(prefix, "/form"), Middleware(formController(prefix), cfg))
	mux.Handle(join(prefix, "/health"), Middleware(healthController, cfg))

	image := ImageMiddleware(cfg, resolver)
	mux.Handle(join(prefix, "/resize"), image(img.Resize))
	mux.Handle(join(prefix, "/fit"), image(img.Fit))
	mux.Handle(join(prefix, "/enlarge"), image(img.Enlarge))
	mux.Handle(join(prefix, "/extract"), image(img.Extract))
	mux.Handle(join(prefix, "/crop"), image(img.Crop))
	mux.Handle(join(prefix, "/smartcrop"), image(img.SmartCrop))
	mux.Handle(join(prefix, "/rotate"), image(img.Rotate))
	mux.Handle(join(prefix, "/autorotate"), image(img.AutoRotate))
	mux.Handle(join(prefix, "/flip"), image(img.Flip))
	mux.Handle(join(prefix, "/flop"), image(img.Flop))
	mux.Handle(join(prefix, "/thumbnail"), image(img.Thumbnail))
	pathThumbnail := Middleware(pathThumbnailController(cfg, resolver), cfg)
	if cfg.EnableURLSignature {
		pathThumbnail = validateURLSignature(pathThumbnail, cfg)
	}
	mux.Handle(thumbnailPathPattern(prefix), pathThumbnail)
	mux.Handle(join(prefix, "/zoom"), image(img.Zoom))
	mux.Handle(join(prefix, "/convert"), image(img.Convert))
	mux.Handle(join(prefix, "/watermark"), image(img.Watermark))
	mux.Handle(join(prefix, "/watermarkimage"), image(img.WatermarkImage))
	mux.Handle(join(prefix, "/info"), image(img.Info))
	mux.Handle(join(prefix, "/blur"), image(img.GaussianBlur))
	mux.Handle(join(prefix, "/pipeline"), image(img.Pipeline))

	// Metadata endpoint (supports images + videos)
	metaHandler := Middleware(metadataController(cfg, resolver), cfg)
	if cfg.EnableURLSignature {
		metaHandler = validateURLSignature(metaHandler, cfg)
	}
	mux.Handle(join(prefix, "/metadata"), metaHandler)

	// Frame extraction endpoint (video → JPEG)
	frameHandler := Middleware(frameController(cfg, resolver), cfg)
	if cfg.EnableURLSignature {
		frameHandler = validateURLSignature(frameHandler, cfg)
	}
	mux.Handle(join(prefix, "/frame"), frameHandler)

	return mux
}
