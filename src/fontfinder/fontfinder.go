package fontfinder

import (
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/adrg/sysfont"
)

type FontPathInfo struct {
	Regular    string
	BoldItalic string
	Italic     string
	Bold       string
}

type fontSearchInfo struct {
	handle    *sysfont.Font
	nameWords []string
	// Style words is not actually styles. But extracted from filename
	// and removed unneeded stuff and splitted to words. On very wide
	// range of the fonts this gives correct hints of the style.
	// But for the fonts which filename has no info (like consola, consolai)
	// we are using the font name if the name is in the sysfont data.
	// Otherwise this package wouldn't help you.
	styleWords   []string
	baseNameWExt string
	hasRegular   bool
	hasItalic    bool
	hasBold      bool
}

var (
	// List of installed system fonts.
	systemFontList      []*sysfont.Font
	systemFontListReady int32

	// Add more if you know any other filename used in
	// font names. All characters must be lowercase.
	regularStrings = []string{"regular", "normal"}
	italicStrings  = []string{"italic", "oblique", "slanted", "it"}
	boldStrings    = []string{"bold"}
)

func init() {

	// On some systems (Windows 10) which has many fonts, this function takes so long.
	// Because of this we are doing this in initilization and in another
	// goroutine. Updates systemFontListReady value to true when finished.
	// And Find() will wait for this to be done only for first time.

	go func() {
		if systemFontList == nil {
			finder := sysfont.NewFinder(&sysfont.FinderOpts{
				Extensions: []string{".ttf", ".otf"},
			})
			systemFontList = finder.List()
		}
		atomic.StoreInt32(&systemFontListReady, 1)
	}()
}

func Find(name string) FontPathInfo {
	for atomic.LoadInt32(&systemFontListReady) != 1 {
		time.Sleep(time.Microsecond)
	}
	return find(name)
}

func find(name string) FontPathInfo {

	fonts := []fontSearchInfo{}

	for _, f := range systemFontList {
		if fontContains(f, name) {
			base := filepath.Base(f.Filename)
			baseWithoutExt := strings.Replace(base, filepath.Ext(base), "", 1)
			styles := strings.Replace(baseWithoutExt, name, "", 1)
			fonts = append(fonts, fontSearchInfo{
				handle:       f,
				nameWords:    SplitWords(f.Name),
				styleWords:   SplitWords(styles),
				baseNameWExt: baseWithoutExt,
			})
		}
	}

	for i := 0; i < len(fonts); i++ {
		f := &fonts[i]
		f.hasRegular = fontHasStyle(f, regularStrings)
		f.hasItalic = fontHasStyle(f, italicStrings)
		f.hasBold = fontHasStyle(f, boldStrings)
	}

	// Sort fonts according to file name lengths in descending order.
	sortFileNameLen(&fonts)

	info := FontPathInfo{}
	regularFounded := false
	for _, f := range fonts {
		// Order is important here.
		if f.hasItalic && f.hasBold {
			info.BoldItalic = f.handle.Filename
		} else if f.hasItalic {
			info.Italic = f.handle.Filename
		} else if f.hasBold {
			info.Bold = f.handle.Filename
		} else if f.hasRegular {
			info.Regular = f.handle.Filename
			// If a font has 'Regular' string, it is the regular.
			// No look for others. If no font has 'Regular' or 'Normal'
			// then the font has smallest filename length and has no
			// italic or bold will be the regular.
			regularFounded = true
		} else if !regularFounded {
			info.Regular = f.handle.Filename
		}
	}

	return info
}

func fontHasStyle(info *fontSearchInfo, stylenames []string) bool {
	for _, s := range stylenames {
		// Split font name to words and look for exact match.
		for _, w := range info.nameWords {
			if s == strings.ToLower(w) {
				return true
			}
		}
		// Do the same thing for the stylenames extracted from filename.
		for _, w := range info.styleWords {
			if s == strings.ToLower(w) {
				return true
			}
		}
	}
	return false
}

func sortFileNameLen(fonts *[]fontSearchInfo) {
	sort.Slice(*fonts, func(i, j int) bool {
		fi := (*fonts)[i]
		fj := (*fonts)[j]
		return len(fi.baseNameWExt) > len(fj.baseNameWExt)
	})
}

// Splits string to words according to delimiters and casing.
// Example:
//  This:           "HelloWorld_from-Turkey"
//  Turns to this:  [Hello, World, from, Turkey]
func SplitWords(str string) []string {
	// CamelCase pascalCase and "-_ "
	arr := []string{}
	runes := []rune(str)
	index := 0
	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '-', '_', ' ', '.', ',':
			word := runes[index:i]
			if len(word) > 0 {
				arr = append(arr, string(word))
				index = i + 1
			}
		default:
			if i < len(runes)-1 {
				curUpper := unicode.IsUpper(runes[i])
				nexUpper := unicode.IsUpper(runes[i+1])
				if curUpper && !nexUpper {
					// Like "Ca"
					word := runes[index:i]
					if len(word) > 0 {
						arr = append(arr, string(word))
						index = i
					}
				} else if !curUpper && nexUpper {
					// Like "aC"
					word := runes[index : i+1]
					if len(word) > 0 {
						arr = append(arr, string(word))
						index = i + 1
					}
				}
			}
		}
	}
	word := runes[index:]
	if len(word) > 0 {
		arr = append(arr, string(word))
	}
	return arr
}

func fontContains(f *sysfont.Font, str string) bool {
	return strings.Contains(f.Name, str) ||
		strings.Contains(f.Family, str) ||
		strings.Contains(f.Filename, str)
}
