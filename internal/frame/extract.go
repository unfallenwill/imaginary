//go:build cgo && ffmpeg

package frame

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"

	"github.com/asticode/go-astiav"

	"github.com/h2non/imaginary/internal/avio"
	img "github.com/h2non/imaginary/internal/image"
)

// FrameResult holds the extracted video frame as a JPEG image.
type FrameResult struct {
	Body []byte
	Mime string
}

// Extract decodes a single video frame at the given time offset and returns it
// as a JPEG image with the video's native dimensions.
func Extract(buf []byte, timeSeconds float64) (FrameResult, error) {
	// 1. Open format context with memory I/O
	formatCtx := astiav.AllocFormatContext()
	if formatCtx == nil {
		return FrameResult{}, img.NewError("Cannot allocate format context", img.KindProcessing)
	}
	defer formatCtx.Free()

	ioCtx, err := avio.NewMemoryIOContext(buf)
	if err != nil {
		return FrameResult{}, img.WrapError("Cannot create IO context", img.KindProcessing, err)
	}
	defer ioCtx.Free()

	formatCtx.SetPb(ioCtx)

	if err := formatCtx.OpenInput("", nil, nil); err != nil {
		return FrameResult{}, img.WrapError("Cannot open video", img.KindProcessing, err)
	}
	defer formatCtx.CloseInput()

	if err := formatCtx.FindStreamInfo(nil); err != nil {
		return FrameResult{}, img.WrapError("Cannot find stream info", img.KindProcessing, err)
	}

	// 2. Find best video stream
	videoStream, suggestedCodec, err := formatCtx.FindBestStream(astiav.MediaTypeVideo, -1, -1)
	if err != nil {
		return FrameResult{}, img.NewError("No video stream found", img.KindUnsupportedMedia)
	}

	// 3. Open codec context
	decoder := suggestedCodec
	if decoder == nil {
		decoder = astiav.FindDecoder(videoStream.CodecParameters().CodecID())
	}
	if decoder == nil {
		return FrameResult{}, img.NewError("Cannot find decoder for video stream", img.KindUnsupportedMedia)
	}

	codecCtx := astiav.AllocCodecContext(decoder)
	if codecCtx == nil {
		return FrameResult{}, img.NewError("Cannot allocate codec context", img.KindProcessing)
	}
	defer codecCtx.Free()

	if err := videoStream.CodecParameters().ToCodecContext(codecCtx); err != nil {
		return FrameResult{}, img.WrapError("Cannot copy codec parameters", img.KindProcessing, err)
	}

	if err := codecCtx.Open(decoder, nil); err != nil {
		return FrameResult{}, img.WrapError("Cannot open codec", img.KindProcessing, err)
	}

	// 4. Seek to requested timestamp (if not zero)
	if timeSeconds > 0 {
		tb := videoStream.TimeBase()
		targetTs := int64(timeSeconds / tb.Float64())

		if err := formatCtx.SeekFrame(videoStream.Index(), targetTs, astiav.NewSeekFlags(astiav.SeekFlagBackward)); err != nil {
			// Seek failed — try decoding from the start instead
			_ = formatCtx.SeekFrame(-1, 0, astiav.NewSeekFlags(astiav.SeekFlagBackward))
		}

		// Flush the decoder's internal buffers after seeking by sending a
		// NULL packet. go-astiav v0.41.0 does not wrap avcodec_flush_buffers,
		// so this is the standard approach to reset decoder state.
		_ = codecCtx.SendPacket(nil)
	}

	// 5. Read packets and decode until we get a frame
	packet := astiav.AllocPacket()
	if packet == nil {
		return FrameResult{}, img.NewError("Cannot allocate packet", img.KindProcessing)
	}
	defer packet.Free()

	decodedFrame := astiav.AllocFrame()
	if decodedFrame == nil {
		return FrameResult{}, img.NewError("Cannot allocate frame", img.KindProcessing)
	}
	defer decodedFrame.Free()

	streamIndex := videoStream.Index()
	var gotFrame bool
	const maxDecodeAttempts = 100

	for i := 0; i < maxDecodeAttempts; i++ {
		if err := formatCtx.ReadFrame(packet); err != nil {
			if err == io.EOF {
				break
			}
			return FrameResult{}, img.WrapError("Error reading packet", img.KindProcessing, err)
		}

		if packet.StreamIndex() != streamIndex {
			packet.Unref()
			continue
		}

		if err := codecCtx.SendPacket(packet); err != nil {
			packet.Unref()
			return FrameResult{}, img.WrapError("Error sending packet to decoder", img.KindProcessing, err)
		}
		packet.Unref()

		decodedFrame.Unref()
		if err := codecCtx.ReceiveFrame(decodedFrame); err != nil {
			continue
		}

		gotFrame = true
		break
	}

	if !gotFrame {
		return FrameResult{}, img.NewError("Cannot decode video frame", img.KindProcessing)
	}

	// 6. Convert pixel format to RGBA via SWSContext
	dstWidth := decodedFrame.Width()
	dstHeight := decodedFrame.Height()

	swsCtx, err := astiav.CreateSoftwareScaleContext(
		dstWidth, dstHeight, decodedFrame.PixelFormat(),
		dstWidth, dstHeight, astiav.PixelFormatRgba,
		astiav.NewSoftwareScaleContextFlags(astiav.SoftwareScaleContextFlagBilinear),
	)
	if err != nil {
		return FrameResult{}, img.WrapError("Cannot create scale context", img.KindProcessing, err)
	}
	defer swsCtx.Free()

	dstFrame := astiav.AllocFrame()
	if dstFrame == nil {
		return FrameResult{}, img.NewError("Cannot allocate destination frame", img.KindProcessing)
	}
	defer dstFrame.Free()

	if err := swsCtx.ScaleFrame(decodedFrame, dstFrame); err != nil {
		return FrameResult{}, img.WrapError("Cannot scale frame", img.KindProcessing, err)
	}

	// 7. Convert to Go image and encode as JPEG
	rgba := image.NewNRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	if err := dstFrame.Data().ToImage(rgba); err != nil {
		return FrameResult{}, img.WrapError("Cannot convert frame to image", img.KindProcessing, err)
	}

	var jpegBuf bytes.Buffer
	if err := jpeg.Encode(&jpegBuf, rgba, &jpeg.Options{Quality: 90}); err != nil {
		return FrameResult{}, img.WrapError("Cannot encode JPEG", img.KindProcessing, err)
	}

	return FrameResult{
		Body: jpegBuf.Bytes(),
		Mime: "image/jpeg",
	}, nil
}
