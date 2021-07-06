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
	handle     *sysfont.Font
	basename   string
	styles     string
	hasRegular bool
	hasItalic  bool
	hasBold    bool
}

var (
	// List of installed system fonts.
	systemFontList      []*sysfont.Font
	systemFontListReady int32

	regularStrings = []string{"regular", "normal"}
	italicStrings  = []string{"italic", "oblique", "slanted", "it"}
	boldStrings    = []string{"bold"}
)

func init() {

	// On some system which has many fonts, this function takes so long.
	// Because of this we are doing this in initilization and in another
	// goroutine. Updates systemFontListReady value to true when finished.
	// And Find() will wait for this to be done.

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
			basename := filepath.Base(f.Filename)
			baseWithoutExt := strings.Replace(basename, filepath.Ext(basename), "", 1)
			styles := strings.Replace(baseWithoutExt, name, "", 1)
			fonts = append(fonts, fontSearchInfo{
				handle:   f,
				basename: basename,
				styles:   strings.ToLower(styles),
			})
		}
	}

	// Remove same words in style names.
	styles := []string{}
	for _, f := range fonts {
		styles = append(styles, f.styles)
	}

	RemoveSameWords(&styles)
	for i := range fonts {
		fonts[i].styles = styles[i]
	}

	for i := 0; i < len(fonts); i++ {
		f := &fonts[i]
		for _, s := range regularStrings {
			if fontNameContains(f.handle, s) || strings.Contains(f.styles, s) {
				f.hasRegular = true
				break
			}
		}
		for _, s := range italicStrings {
			if fontNameContains(f.handle, s) || strings.Contains(f.styles, s) {
				f.hasItalic = true
				break
			}
		}
		for _, s := range boldStrings {
			if fontNameContains(f.handle, s) || strings.Contains(f.styles, s) {
				f.hasBold = true
				break
			}
		}
	}

	// Sort fonts according to file name length.
	sortFileNameLen(&fonts)

	info := FontPathInfo{}

	regularFounded := false
	for _, f := range fonts {
		if f.hasItalic && f.hasBold {
			info.BoldItalic = f.handle.Filename
		} else if f.hasItalic {
			info.Italic = f.handle.Filename
		} else if f.hasBold {
			info.Bold = f.handle.Filename
		} else if f.hasRegular {
			info.Regular = f.handle.Filename
			regularFounded = true
		} else if !regularFounded {
			info.Regular = f.handle.Filename
		}
	}

	return info
}

func sortFileNameLen(fonts *[]fontSearchInfo) {
	sort.Slice(*fonts, func(i, j int) bool {
		fi := (*fonts)[i]
		fj := (*fonts)[j]
		return len(fi.handle.Filename) > len(fj.handle.Filename)
	})
}

// Finds and removes all same words in array of strings.
// TODO: optimize
func RemoveSameWords(strs *[]string) {
	strwords := [][]string{}
	for _, s := range *strs {
		strwords = append(strwords, SplitWords(s))
	}
	for i, s1 := range strwords {
		for _, w1 := range s1 {
			total := 0
			for j, s2 := range strwords {
				if i != j {
					for _, w2 := range s2 {
						if w1 == w2 {
							total++
						}
					}
				}
			}
			if total == len(strwords)-1 {
				for i := 0; i < len(*strs); i++ {
					(*strs)[i] = strings.Replace((*strs)[i], w1, "", 1)
				}
			}
		}
	}
}

// Splits string to words according to delimiters and casing.
// Example:
//	This: 			"HelloWorld_from-Turkey"
//  Turns to this: 	[Hello, World, from, Turkey]
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

func fontNameContains(f *sysfont.Font, str string) bool {
	return strings.Contains(strings.ToLower(f.Name), str) ||
		strings.Contains(strings.ToLower(f.Family), str)
}

func fontContains(f *sysfont.Font, str string) bool {
	return strings.Contains(f.Name, str) ||
		strings.Contains(f.Family, str) ||
		strings.Contains(f.Filename, str)
}
