package fontkit

import (
	"fmt"
	"os"

	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// FontKit is a struct that holds different styles of same font family
type Font struct {
	handle   *sfnt.Font
	buffer   sfnt.Buffer
	filePath string
}

func CreateFontFromFile(pathToFile string) (*Font, error) {
	fileData, err := os.ReadFile(pathToFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file: %s\n", err)
	}
	font, err := CreateFontFromMem(fileData)
	if err != nil {
		return nil, err
	}
	font.filePath = pathToFile
	return font, nil
}

func CreateFontFromMem(data []byte) (*Font, error) {
	font := new(Font)
	var err error
	font.handle, err = opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse font data: %s\n", err)
	}
	return font, nil
}

func (font *Font) FilePath() string {
	return font.filePath
}

func (font *Font) FamilyName() (string, error) {
	name, err := font.handle.Name(&font.buffer, sfnt.NameIDFamily)
	if err != nil {
		return "", err
	}
	return name, nil
}

// ContainsGlyph returns the whether font contains the given glyph.
func (font *Font) ContainsGlyph(char rune) bool {
	i, err := font.handle.GlyphIndex(&font.buffer, char)
	return i != 0 && err == nil
}
