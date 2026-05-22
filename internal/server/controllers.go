package server

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/h2non/bimg"
	"github.com/h2non/filetype"

	"github.com/h2non/imaginary/internal/config"
	img "github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
	"github.com/h2non/imaginary/internal/version"
)

func indexController(o config.ServerOptions) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path.Join(o.PathPrefix, "/") {
			img.ErrorReply(r, w, img.ErrNotFound, config.ServerOptions{})
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

func imageController(o config.ServerOptions, resolver *source.Resolver, operation img.Operation) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var imageSource = resolver.Match(req)
		if imageSource == nil {
			img.ErrorReply(req, w, img.ErrMissingImageSource, o)
			return
		}

		buf, err := imageSource.GetImage(req)
		if err != nil {
			if xerr, ok := err.(img.Error); ok {
				img.ErrorReply(req, w, xerr, o)
			} else {
				img.ErrorReply(req, w, img.NewError(err.Error(), http.StatusBadRequest), o)
			}
			return
		}

		if len(buf) == 0 {
			img.ErrorReply(req, w, img.ErrEmptyBody, o)
			return
		}

		imageHandler(w, req, buf, operation, o)
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

func imageHandler(w http.ResponseWriter, r *http.Request, buf []byte, operation img.Operation, o config.ServerOptions) {
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
		img.ErrorReply(r, w, img.ErrUnsupportedMedia, o)
		return
	}

	opts, err := img.BuildParamsFromQuery(r.URL.Query())
	if err != nil {
		img.ErrorReply(r, w, img.NewError("Error while processing parameters, "+err.Error(), http.StatusBadRequest), o)
		return
	}

	vary := ""
	if opts.Type == "auto" {
		opts.Type = determineAcceptMimeType(r.Header.Get("Accept"))
		vary = "Accept"
	} else if opts.Type != "" && img.ImageType(opts.Type) == 0 {
		img.ErrorReply(r, w, img.ErrOutputFormat, o)
		return
	}

	sizeInfo, err := bimg.Size(buf)

	if err != nil {
		img.ErrorReply(r, w, img.NewError("Error while processing the image: "+err.Error(), http.StatusBadRequest), o)
		return
	}

	imgResolution := float64(sizeInfo.Width) * float64(sizeInfo.Height)

	if (imgResolution / 1000000) > o.MaxAllowedPixels {
		img.ErrorReply(r, w, img.ErrResolutionTooBig, o)
		return
	}

	image, err := operation.Run(buf, opts)
	if err != nil {
		if vary != "" {
			w.Header().Set("Vary", vary)
		}
		img.ErrorReply(r, w, img.NewError("Error while processing the image: "+err.Error(), http.StatusBadRequest), o)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(image.Body)))
	w.Header().Set("Content-Type", image.Mime)
	if image.Mime != "application/json" && o.ReturnSize {
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

func formController(o config.ServerOptions) func(w http.ResponseWriter, r *http.Request) {
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
		</form>`, path.Join(o.PathPrefix, form.name), path.Join(o.PathPrefix, form.method), form.args)
		}

		html += "</body></html>"

		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(html))
	}
}
