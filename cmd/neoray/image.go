package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/opengl"
	"github.com/hismailbulut/Neoray/pkg/window"
	"golang.org/x/image/bmp"
	"golang.org/x/image/webp"
)

type ImageViewer struct {
	hidden    bool
	imageChan chan string
	window    *window.Window
	texture   opengl.Texture
	buffer    *opengl.VertexBuffer
}

func NewImageViewer(window *window.Window) *ImageViewer {
	return &ImageViewer{
		hidden:    true,
		imageChan: make(chan string, 4),
		window:    window,
		texture:   window.GL().CreateTexture(64, 64), // Temporary size
		buffer:    window.GL().CreateVertexBuffer(1),
	}
}

func (viewer *ImageViewer) LoadImageFromFile(path string) (*image.RGBA, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// We load image by it's extension
	format := filepath.Ext(path)[1:] // Remove dot
	// Load image
	var img image.Image
	switch format {
	case "png":
		img, err = png.Decode(file)
		if err != nil {
			return nil, err
		}
	case "jpg", "jpeg":
		img, err = jpeg.Decode(file)
		if err != nil {
			return nil, err
		}
	case "gif":
		img, err = gif.Decode(file)
		if err != nil {
			return nil, err
		}
	case "webp":
		img, err = webp.Decode(file)
		if err != nil {
			return nil, err
		}
	case "bmp":
		img, err = bmp.Decode(file)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Unsupported image format: %s", format)
	}
	// Check image type and convert to RGBA if needed
	var imgRGBA *image.RGBA
	switch img := img.(type) {
	case *image.RGBA:
		imgRGBA = img
	default:
		imgRGBA = image.NewRGBA(img.Bounds())
		draw.Draw(imgRGBA, img.Bounds(), img, image.Point{}, draw.Over)
	}
	// Image should not has transparent pixels
	// Traverse through image and set every pixel alpha to 255
	for i := 3; i < len(imgRGBA.Pix); i += 4 {
		imgRGBA.Pix[i] = 255
	}
	return imgRGBA, nil
}

func (viewer *ImageViewer) SetImage(path string) error {
	imgRGBA, err := viewer.LoadImageFromFile(path)
	if err != nil {
		return fmt.Errorf("Could not load image: %s", err)
	}
	// Resize opengl texture
	viewer.texture.Bind()
	viewer.texture.Resize(imgRGBA.Rect.Dx(), imgRGBA.Rect.Dy())
	// Draw image to texture
	dest := common.Rectangle[int]{
		X: 0,
		Y: 0,
		W: imgRGBA.Bounds().Dx(),
		H: imgRGBA.Bounds().Dy(),
	}
	viewer.texture.Draw(imgRGBA, dest)
	// This is always same for every texture
	viewer.buffer.SetIndexTex1(0, common.Rectangle[float32]{X: 0, Y: 0, W: 1, H: 1})
	return nil
}

func (viewer *ImageViewer) Show() {
	if !viewer.hidden {
		return
	}
	viewer.hidden = false
	MarkDraw()
}

func (viewer *ImageViewer) Hide() {
	if viewer.hidden {
		return
	}
	viewer.hidden = true
	MarkRender()
}

func (viewer *ImageViewer) IsVisible() bool {
	return !viewer.hidden
}

func (viewer *ImageViewer) Update() {
	if len(viewer.imageChan) > 0 {
		err := viewer.SetImage(<-viewer.imageChan)
		if err != nil {
			Editor.nvim.EchoError("%v", err)
		} else {
			viewer.Show()
		}
	}
}

func (viewer *ImageViewer) Draw() {
	if viewer.hidden {
		return
	}
	// We fit texture width and height to screen area
	// And keep aspect ratio while doing this
	w := float32(viewer.window.Size().Width())
	h := float32(viewer.window.Size().Height())
	imgW := float32(viewer.texture.Size().Width())
	imgH := float32(viewer.texture.Size().Height())
	wRatio := w / imgW
	hRatio := h / imgH
	// Fit image
	if wRatio < hRatio {
		// Fit width and keep ratio
		ratio := imgH / imgW
		imgW = w
		imgH = imgW * ratio
	} else {
		// Fit height and keep ratio
		ratio := imgW / imgH
		imgH = h
		imgW = imgH * ratio
	}
	// Apply padding
	const paddingRatio = 16
	var paddingX float32 = imgW / paddingRatio
	var paddingY float32 = imgH / paddingRatio
	position := common.Rectangle[float32]{
		X: paddingX + (w/2 - imgW/2),
		Y: paddingY + (h/2 - imgH/2),
		W: imgW - 2*paddingX,
		H: imgH - 2*paddingY,
	}
	viewer.buffer.SetIndexPos(0, position)
}

func (viewer *ImageViewer) Render() {
	if viewer.hidden {
		return
	}
	viewer.texture.Bind()
	viewer.buffer.Bind()
	viewer.buffer.Update()
	// viewer.buffer.SetProjection(viewer.window.Viewport().ToF32())
	viewer.buffer.Render()
}

func (viewer *ImageViewer) Destroy() {
	// Delete opengl texture
	viewer.texture.Delete()
	// Destroy opengl vertex buffer
	viewer.buffer.Destroy()
}
