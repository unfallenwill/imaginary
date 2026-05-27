package server

import (
	"bytes"
	"context"
	"encoding/json"
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

	framepkg "github.com/h2non/imaginary/internal/frame"
	img "github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/metadata"
	"github.com/h2non/imaginary/internal/source"
	"github.com/h2non/imaginary/internal/version"
)

const (
	megaPixel               = 1_000_000
	defaultMaxWatermarkSize = 1 << 20 // 1MB
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

		buf, err := imageSource.GetImage(req.Context(), req)
		if err != nil {
			ErrorReply(w, req, err, cfg.Error)
			return
		}

		if len(buf) == 0 {
			ErrorReply(w, req, img.ErrEmptyBody, cfg.Error)
			return
		}

		imageHandler(w, req, buf, operation, cfg)
	}
}

// metadataController handles metadata extraction requests for both images and videos.
// It resolves the media source, detects the media type, and dispatches to the
// appropriate metadata extractor. Unlike imageController, it does not perform
// image-specific MIME validation, resolution checks, or watermark resolution.
func metadataController(cfg Config, resolver *source.Resolver) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		imageSource := resolver.Match(req)
		if imageSource == nil {
			ErrorReply(w, req, img.ErrMissingImageSource, cfg.Error)
			return
		}

		buf, err := imageSource.GetImage(req.Context(), req)
		if err != nil {
			ErrorReply(w, req, err, cfg.Error)
			return
		}

		if len(buf) == 0 {
			ErrorReply(w, req, img.ErrEmptyBody, cfg.Error)
			return
		}

		mediaType := detectMediaType(buf)

		result, err := metadata.Extract(buf, mediaType)
		if err != nil {
			ErrorReply(w, req, err, cfg.Error)
			return
		}

		w.Header().Set("Content-Type", result.Mime)
		w.Header().Set("Content-Length", strconv.Itoa(len(result.Body)))
		_, _ = w.Write(result.Body)
	}
}

// frameController handles video frame extraction requests.
// It resolves the media source, validates it as video, and extracts a single
// frame at the requested time offset as a JPEG image.
func frameController(cfg Config, resolver *source.Resolver) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		imageSource := resolver.Match(req)
		if imageSource == nil {
			ErrorReply(w, req, img.ErrMissingImageSource, cfg.Error)
			return
		}

		buf, err := imageSource.GetImage(req.Context(), req)
		if err != nil {
			ErrorReply(w, req, err, cfg.Error)
			return
		}

		if len(buf) == 0 {
			ErrorReply(w, req, img.ErrEmptyBody, cfg.Error)
			return
		}

		timeSeconds := 0.0
		if t := req.URL.Query().Get("time"); t != "" {
			timeSeconds, err = strconv.ParseFloat(t, 64)
			if err != nil || timeSeconds < 0 {
				ErrorReply(w, req, img.New(img.KindInvalidParam, "Invalid time parameter: must be a non-negative number"), cfg.Error)
				return
			}
		}

		mediaType := detectMediaType(buf)
		if mediaType != metadata.MediaTypeVideo {
			ErrorReply(w, req, img.New(img.KindUnsupportedMedia, "Frame extraction requires a video input"), cfg.Error)
			return
		}

		result, err := framepkg.Extract(buf, timeSeconds)
		if err != nil {
			ErrorReply(w, req, err, cfg.Error)
			return
		}

		w.Header().Set("Content-Type", result.Mime)
		w.Header().Set("Content-Length", strconv.Itoa(len(result.Body)))
		_, _ = w.Write(result.Body)
	}
}

// detectMediaType determines whether the byte buffer contains an image or video.
// It checks image MIME types first (via bimg support), then falls back to
// video detection via MIME prefix and magic byte signatures.
func detectMediaType(buf []byte) metadata.MediaType {
	mime := detectImageMimeType(buf)
	if img.IsImageMimeTypeSupported(mime) {
		return metadata.MediaTypeImage
	}

	if strings.HasPrefix(mime, "video/") {
		return metadata.MediaTypeVideo
	}

	if isVideoByMagicBytes(buf) {
		return metadata.MediaTypeVideo
	}

	kind, err := filetype.Get(buf)
	if err == nil {
		if strings.HasPrefix(kind.MIME.Value, "video/") {
			return metadata.MediaTypeVideo
		}
	}

	return metadata.MediaTypeUnknown
}

// isVideoByMagicBytes checks for common video container format signatures.
func isVideoByMagicBytes(buf []byte) bool {
	if len(buf) < 12 {
		return false
	}

	// MP4/MOV/M4A: ftyp box at offset 4
	if len(buf) > 8 && string(buf[4:8]) == "ftyp" {
		return true
	}

	// Matroska/WebM: EBML header
	if bytes.HasPrefix(buf, []byte{0x1A, 0x45, 0xDF, 0xA3}) {
		return true
	}

	// AVI: RIFF....AVI
	if bytes.HasPrefix(buf, []byte("RIFF")) && len(buf) > 11 && string(buf[8:11]) == "AVI" {
		return true
	}

	// FLV: Flash Video
	if buf[0] == 'F' && buf[1] == 'L' && buf[2] == 'V' {
		return true
	}

	// MPEG-TS: 0x47 sync byte pattern
	if buf[0] == 0x47 && len(buf) > 188 && buf[188] == 0x47 {
		return true
	}

	return false
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

// detectImageMimeType determines the MIME type of raw image bytes.
// It uses http.DetectContentType as the primary detector, falls back to
// filetype matching for application/octet-stream, and checks for SVG
// in text/plain responses.
func detectImageMimeType(buf []byte) string {
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

	return mimeType
}

// parseOutputType resolves the output image type from query parameters.
// If type is "auto", it negotiates from the Accept header and returns "Accept"
// for the Vary response header. Returns an error if the explicitly requested
// type is unsupported.
func parseOutputType(opts *img.ImageOptions, accept string) (string, error) {
	if opts.Type == "auto" {
		opts.Type = determineAcceptMimeType(accept)
		return "Accept", nil
	}
	if opts.Type != "" && img.ImageType(opts.Type) == 0 {
		return "", img.ErrOutputFormat
	}
	return "", nil
}

// checkResolution validates that image dimensions do not exceed the configured
// megapixel limit. Returns ErrResolutionTooBig if the image is too large, or a
// wrapped error if the image cannot be decoded.
func checkResolution(buf []byte, maxPixels float64) error {
	if maxPixels <= 0 {
		return nil
	}
	sizeInfo, err := bimg.Size(buf)
	if err != nil {
		return img.Wrap(img.KindProcessing, "cannot determine image size", err)
	}
	imgResolution := float64(sizeInfo.Width) * float64(sizeInfo.Height)
	if (imgResolution / megaPixel) > maxPixels {
		return img.ErrResolutionTooBig
	}
	return nil
}

// resolveWatermarks downloads remote watermark images for both single-image
// and pipeline watermark operations, storing fetched bytes into opts.
// This ensures the image domain layer never performs I/O.
func resolveWatermarks(ctx context.Context, opts *img.ImageOptions, client *http.Client, maxSize int) error {
	if opts.Image != "" && len(opts.ImageBytes) == 0 {
		imageBytes, err := fetchRemoteImage(client, ctx, opts.Image, maxSize)
		if err != nil {
			return err
		}
		opts.ImageBytes = imageBytes
	}

	for i := range opts.Operations {
		op := &opts.Operations[i]
		if op.Name == "watermarkImage" && op.Params.Image != "" && len(op.Params.ImageBytes) == 0 {
			imageBytes, err := fetchRemoteImage(client, ctx, op.Params.Image, maxSize)
			if err != nil {
				return err
			}
			op.Params.ImageBytes = imageBytes
		}
	}

	return nil
}

func imageHandler(w http.ResponseWriter, r *http.Request, buf []byte, operation img.Operation, cfg Config) {
	mimeType := detectImageMimeType(buf)
	if !img.IsImageMimeTypeSupported(mimeType) {
		ErrorReply(w, r, img.ErrUnsupportedMedia, cfg.Error)
		return
	}

	opts, err := BuildParamsFromQuery(map[string][]string(r.URL.Query()))
	if err != nil {
		ErrorReply(w, r, img.Wrap(img.KindInvalidParam, "invalid image parameters", err), cfg.Error)
		return
	}

	vary, err := parseOutputType(&opts, r.Header.Get("Accept"))
	if err != nil {
		ErrorReply(w, r, err, cfg.Error)
		return
	}

	if err := checkResolution(buf, cfg.MaxAllowedPixels); err != nil {
		ErrorReply(w, r, err, cfg.Error)
		return
	}

	if err := resolveWatermarks(r.Context(), &opts, cfg.RemoteClient, cfg.MaxAllowedSize); err != nil {
		ErrorReply(w, r, err, cfg.Error)
		return
	}

	image, err := operation.Run(buf, opts)
	if err != nil {
		if vary != "" {
			w.Header().Set("Vary", vary)
		}
		ErrorReply(w, r, err, cfg.Error)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(image.Body)))
	w.Header().Set("Content-Type", image.Mime)
	if image.Mime != "application/json" && cfg.ReturnSize {
		if meta, err := bimg.Metadata(image.Body); err == nil {
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
			{"Media metadata (image/video)", "metadata", ""},
			{"Video frame extraction", "frame", "time=1.5"},
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

// fetchRemoteImage downloads an image from a URL with proper context propagation,
// timeout, and size limits. This is the only place in the server where outbound
// HTTP requests for watermark images are made.
func fetchRemoteImage(client *http.Client, ctx context.Context, imageURL string, maxAllowedSize int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil) // #nosec G704 -- imageURL is validated by the allowed-origins whitelist in the source layer
	if err != nil {
		return nil, img.Wrap(img.KindInvalidParam, "invalid watermark URL", err)
	}

	resp, err := client.Do(req) // #nosec G704 -- see above
	if err != nil {
		return nil, img.Wrap(img.KindUpstream, "fetch failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, img.New(img.KindUpstream, fmt.Sprintf("remote returned status %d", resp.StatusCode))
	}

	var reader io.Reader = resp.Body
	limit := defaultMaxWatermarkSize
	if maxAllowedSize > 0 {
		limit = maxAllowedSize
	}
	reader = io.LimitReader(reader, int64(limit)+1)

	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, img.Wrap(img.KindUpstream, "read body", err)
	}
	if len(buf) > limit {
		return nil, img.ErrContentTooLarge
	}
	if len(buf) == 0 {
		return nil, img.ErrEmptyBody
	}

	return buf, nil
}
