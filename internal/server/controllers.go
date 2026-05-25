package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/h2non/bimg"
	"github.com/h2non/filetype"

	img "github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
	"github.com/h2non/imaginary/internal/version"
)

const (
	megaPixel                = 1_000_000
	defaultMaxWatermarkSize  = 1 << 20 // 1MB
)

func indexController(prefix string, errCfg ErrorConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path.Join(prefix, "/") {
			ErrorReply(w, r, img.ErrNotFound, errCfg)
			return
		}

		body, _ := json.Marshal(version.Versions{
			ImaginaryVersion: version.Version,
			BimgVersion:      bimg.Version,
			VipsVersion:      bimg.VipsVersion,
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

func healthController(w http.ResponseWriter, r *http.Request) {
	health := GetHealthStats()
	body, _ := json.Marshal(health)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func imageController(cfg Config, resolver *source.Resolver, operation img.Operation) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		imageSource := resolver.Match(req)
		if imageSource == nil {
			ErrorReply(w, req, img.ErrMissingImageSource, cfg.Error)
			return
		}

		buf, err := imageSource.GetImage(req)
		if err != nil {
			replyError(w, req, err, img.KindUpstream, cfg)
			return
		}

		if len(buf) == 0 {
			ErrorReply(w, req, img.ErrEmptyBody, cfg.Error)
			return
		}

		imageHandler(w, req, buf, operation, cfg)
	}
}

func determineAcceptMimeType(accept string) string {
	for _, v := range strings.Split(accept, ",") {
		mediaType, _, _ := mime.ParseMediaType(v)
		switch mediaType {
		case "image/webp":
			return "webp"
		case "image/png":
			return "png"
		case "image/jpeg":
			return "jpeg"
		}
	}

	return ""
}

func imageHandler(w http.ResponseWriter, r *http.Request, buf []byte, operation img.Operation, cfg Config) {
	mimeType := http.DetectContentType(buf)

	if mimeType == "application/octet-stream" {
		kind, err := filetype.Get(buf)
		if err == nil && kind.MIME.Value != "" {
			mimeType = kind.MIME.Value
		}
	}

	if strings.Contains(mimeType, "text/plain") && len(buf) > 8 {
		if bimg.IsSVGImage(buf) {
			mimeType = "image/svg+xml"
		}
	}

	if !img.IsImageMimeTypeSupported(mimeType) {
		ErrorReply(w, r, img.ErrUnsupportedMedia, cfg.Error)
		return
	}

	opts, err := img.BuildParamsFromQuery(r.URL.Query())
	if err != nil {
		replyError(w, r, err, img.KindInvalidParam, cfg)
		return
	}

	vary := ""
	if opts.Type == "auto" {
		opts.Type = determineAcceptMimeType(r.Header.Get("Accept"))
		vary = "Accept"
	} else if opts.Type != "" && img.ImageType(opts.Type) == 0 {
		ErrorReply(w, r, img.ErrOutputFormat, cfg.Error)
		return
	}

	sizeInfo, err := bimg.Size(buf)

	if err != nil {
		replyError(w, r, err, img.KindProcessing, cfg)
		return
	}

	imgResolution := float64(sizeInfo.Width) * float64(sizeInfo.Height)

	if (imgResolution / megaPixel) > cfg.MaxAllowedPixels {
		ErrorReply(w, r, img.ErrResolutionTooBig, cfg.Error)
		return
	}

	// Resolve watermark image URL to bytes before entering the pure domain layer.
	// The image package must never perform I/O — all network calls happen here.
	if opts.Image != "" && len(opts.ImageBytes) == 0 {
		imageBytes, err := fetchRemoteImage(cfg.RemoteClient, r.Context(), opts.Image, cfg.MaxAllowedSize)
		if err != nil {
			replyError(w, r, err, img.KindUpstream, cfg)
			return
		}
		opts.ImageBytes = imageBytes
	}

	// Pipeline operations that contain a watermarkImage step also need their
	// image URLs resolved to bytes before execution.
	for i := range opts.Operations {
		if opts.Operations[i].Name == "watermarkImage" && opts.Operations[i].Params.Image != "" {
			imageBytes, err := fetchRemoteImage(cfg.RemoteClient, r.Context(), opts.Operations[i].Params.Image, cfg.MaxAllowedSize)
			if err != nil {
				replyError(w, r, err, img.KindUpstream, cfg)
				return
			}
			opts.Operations[i].Params.ImageBytes = imageBytes
		}
	}

	image, err := operation.Run(buf, opts)
	if err != nil {
		if vary != "" {
			w.Header().Set("Vary", vary)
		}
		replyError(w, r, err, img.KindProcessing, cfg)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(image.Body)))
	w.Header().Set("Content-Type", image.Mime)
	if image.Mime != "application/json" && cfg.ReturnSize {
		meta, err := bimg.Metadata(image.Body)
		if err == nil {
			w.Header().Set("Image-Width", strconv.Itoa(meta.Size.Width))
			w.Header().Set("Image-Height", strconv.Itoa(meta.Size.Height))
		}
	}
	if vary != "" {
		w.Header().Set("Vary", vary)
	}
	_, _ = w.Write(image.Body)
}

func formController(prefix string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		operations := []struct {
			name   string
			method string
			args   string
		}{
			{"Resize", "resize", "width=300&height=200&type=jpeg"},
			{"Force resize", "resize", "width=300&height=200&force=true"},
			{"Crop", "crop", "width=300&quality=95"},
			{"SmartCrop", "crop", "width=300&height=260&quality=95&gravity=smart"},
			{"Extract", "extract", "top=100&left=100&areawidth=300&areaheight=150"},
			{"Enlarge", "enlarge", "width=1440&height=900&quality=95"},
			{"Rotate", "rotate", "rotate=180"},
			{"AutoRotate", "autorotate", "quality=90"},
			{"Flip", "flip", ""},
			{"Flop", "flop", ""},
			{"Thumbnail", "thumbnail", "width=100"},
			{"Zoom", "zoom", "factor=2&areawidth=300&top=80&left=80"},
			{"Color space (black&white)", "resize", "width=400&height=300&colorspace=bw"},
			{"Add watermark", "watermark", "textwidth=100&text=Hello&font=sans%2012&opacity=0.5&color=255,200,50"},
			{"Convert format", "convert", "type=png"},
			{"Image metadata", "info", ""},
			{"Gaussian blur", "blur", "sigma=15.0&minampl=0.2"},
			{"Pipeline (image reduction via multiple transformations)", "pipeline", "operations=%5B%7B%22operation%22:%20%22crop%22,%20%22params%22:%20%7B%22width%22:%20300,%20%22height%22:%20260%7D%7D,%20%7B%22operation%22:%20%22convert%22,%20%22params%22:%20%7B%22type%22:%20%22webp%22%7D%7D%5D"},
		}

		html := "<html><body>"

		for _, form := range operations {
			html += fmt.Sprintf(`
		<h1>%s</h1>
		<form method="POST" action="%s?%s" enctype="multipart/form-data">
		<input type="file" name="file" />
		<input type="submit" value="Upload" />
		</form>`, path.Join(prefix, form.name), path.Join(prefix, form.method), form.args)
		}

		html += "</body></html>"

		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(html))
	}
}

// replyError writes an error reply, transparently passing through *image.Error.
// Non-image errors are wrapped with the given kind.
func replyError(w http.ResponseWriter, r *http.Request, err error, kind img.Kind, cfg Config) {
	var imgErr *img.Error
	if errors.As(err, &imgErr) {
		ErrorReply(w, r, imgErr, cfg.Error)
	} else {
		ErrorReply(w, r, img.WrapError(err.Error(), kind, err), cfg.Error)
	}
}

// fetchRemoteImage downloads an image from a URL with proper context propagation,
// timeout, and size limits. This is the only place in the server where outbound
// HTTP requests for watermark images are made.
func fetchRemoteImage(client *http.Client, ctx context.Context, imageURL string, maxAllowedSize int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, img.WrapError("invalid watermark URL", img.KindInvalidParam, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, img.WrapError("fetch failed", img.KindUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, img.NewError(fmt.Sprintf("remote returned status %d", resp.StatusCode), img.KindUpstream)
	}

	var reader io.Reader = resp.Body
	if maxAllowedSize > 0 {
		reader = io.LimitReader(reader, int64(maxAllowedSize))
	} else {
		reader = io.LimitReader(reader, defaultMaxWatermarkSize)
	}

	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, img.WrapError("read body", img.KindUpstream, err)
	}
	if len(buf) == 0 {
		return nil, img.ErrEmptyBody
	}

	return buf, nil
}
