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
	"github.com/h2non/imaginary/internal/config"
)

func Server(o config.ServerOptions) {
	addr := o.Address + ":" + strconv.Itoa(o.Port)
	handler := NewLog(NewServerMux(o), os.Stdout, o.LogLevel)

	server := &http.Server{
		Addr:           addr,
		Handler:        handler,
		MaxHeaderBytes: 1 << 20,
		ReadTimeout:    time.Duration(o.HTTPReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(o.HTTPWriteTimeout) * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := listenAndServe(server, o); err != nil && err != http.ErrServerClosed {
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

func listenAndServe(s *http.Server, o config.ServerOptions) error {
	if o.CertFile != "" && o.KeyFile != "" {
		return s.ListenAndServeTLS(o.CertFile, o.KeyFile)
	}
	return s.ListenAndServe()
}

func join(o config.ServerOptions, route string) string {
	return path.Join(o.PathPrefix, route)
}

// NewServerMux creates a new HTTP server route multiplexer.
func NewServerMux(o config.ServerOptions) http.Handler {
	mux := http.NewServeMux()

	mux.Handle(join(o, "/"), Middleware(indexController(o), o))
	mux.Handle(join(o, "/form"), Middleware(formController(o), o))
	mux.Handle(join(o, "/health"), Middleware(healthController, o))

	image := ImageMiddleware(o)
	mux.Handle(join(o, "/resize"), image(img.Resize))
	mux.Handle(join(o, "/fit"), image(img.Fit))
	mux.Handle(join(o, "/enlarge"), image(img.Enlarge))
	mux.Handle(join(o, "/extract"), image(img.Extract))
	mux.Handle(join(o, "/crop"), image(img.Crop))
	mux.Handle(join(o, "/smartcrop"), image(img.SmartCrop))
	mux.Handle(join(o, "/rotate"), image(img.Rotate))
	mux.Handle(join(o, "/autorotate"), image(img.AutoRotate))
	mux.Handle(join(o, "/flip"), image(img.Flip))
	mux.Handle(join(o, "/flop"), image(img.Flop))
	mux.Handle(join(o, "/thumbnail"), image(img.Thumbnail))
	mux.Handle(join(o, "/zoom"), image(img.Zoom))
	mux.Handle(join(o, "/convert"), image(img.Convert))
	mux.Handle(join(o, "/watermark"), image(img.Watermark))
	mux.Handle(join(o, "/watermarkimage"), image(img.WatermarkImage))
	mux.Handle(join(o, "/info"), image(img.Info))
	mux.Handle(join(o, "/blur"), image(img.GaussianBlur))
	mux.Handle(join(o, "/pipeline"), image(img.Pipeline))

	return mux
}
