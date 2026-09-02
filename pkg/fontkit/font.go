package fontkit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hismailbulut/Neoray/pkg/logger"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

type Font struct {
	handles   []*sfnt.Font
	buffer    sfnt.Buffer
	file      *os.File
	filePath  string
	faceCache map[FaceParams]*Face
}

func LoadFontFromFile(pathToFile string) (*Font, error) {
	file, err := os.Open(pathToFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file: %s\n", err)
	}
	font, err := LoadFontFromReader(file)
	if err != nil {
		defer file.Close()
		return nil, err
	}
	font.file = file
	font.filePath = pathToFile
	return font, nil
}

func LoadFontFromMem(data []byte) (*Font, error) {
	return LoadFontFromReader(bytes.NewReader(data))
}

func LoadFontFromReader(reader io.ReaderAt) (*Font, error) {
	font := new(Font)
	var err error
	collection, err := opentype.ParseCollectionReaderAt(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %s", err)
	}
	for i := 0; i < collection.NumFonts(); i++ {
		f, err := collection.Font(i)
		if err != nil {
			return nil, fmt.Errorf("failed to parse font: %s", err)
		}
		font.handles = append(font.handles, f)
	}
	font.faceCache = make(map[FaceParams]*Face)
	return font, nil
}

func (f *Font) createFace(params FaceParams, index int) (*Face, error) {
	face, ok := f.faceCache[params]
	if ok {
		return face, nil
	} else {
		face, err := NewFace(f.handles[index], params)
		if err != nil {
			return nil, err
		}
		f.faceCache[params] = face
		return face, nil
	}
}

// This funtion may return a face previously created and used because it caches
// every face in the font and it also caches every image size. Caching and reusing
// makes this library incredibly fast and memory friendly. But creating so many
// faces and drawing multiple size of images every time increases memory usage.
// And this memory never be freed until the font has freed. (This is not leak)
func (f *Font) CreateFace(params FaceParams) (*Face, error) {
	if len(f.handles) == 0 {
		return nil, errors.New("font not initialized")
	}
	return f.createFace(params, 0)
}

// This is same with CreateFace but finds the exact font which contains given glyph
// in a font collection and creates a face from it
func (f *Font) CreateFaceForGlyph(params FaceParams, char rune) (*Face, error) {
	i := f.indexOfFontContainsGlyph(char)
	if i < 0 {
		// This font does not contain this glyph
		return nil, fmt.Errorf("glyph %s not exists in this font", string(char))
	}
	return f.createFace(params, i)
}

func (font *Font) FilePath() string {
	return font.filePath
}

func (font *Font) Name(id sfnt.NameID) string {
	for i := 0; i < len(font.handles); i++ {
		name, err := font.handles[i].Name(&font.buffer, id)
		if err == nil {
			return name
		} else {
			logger.Error("Failed to get font family name:", err)
		}
	}
	return ""
}

func (font *Font) indexOfFontContainsGlyph(char rune) int {
	for i := 0; i < len(font.handles); i++ {
		j, err := font.handles[i].GlyphIndex(&font.buffer, char)
		if j != 0 && err == nil {
			return i
		}
	}
	return -1
}

// ContainsGlyph returns whether this font contains the given glyph.
func (font *Font) ContainsGlyph(char rune) bool {
	return font.indexOfFontContainsGlyph(char) >= 0
}

func (font *Font) Unload() {
	if font.file != nil {
		font.file.Close()
	}
}
