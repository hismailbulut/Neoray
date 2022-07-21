package fontkit

import (
	"fmt"
	"image"

	"github.com/hismailbulut/Neoray/pkg/common"
)

func (face *Face) DrawUnicodeBoxGlyph(char rune, imgSize common.Vector2[int]) *image.RGBA {
	light := face.thickness
	heavy := light * 2

	img := face.cachedImage(imgSize)
	w := float32(imgSize.Width())
	h := float32(imgSize.Height())

	var (
		center      = common.Vec2(w/2, h/2)
		centerUp    = common.Vec2(w/2, 0)
		centerDown  = common.Vec2(w/2, h)
		centerLeft  = common.Vec2(0, h/2)
		centerRight = common.Vec2(w, h/2)
	)

	switch char {
	case 0x2500: // light horizontal line
		drawRectLine(img, centerLeft, centerRight, light, 1)
	case 0x2501: // heavy horizontal line
		drawRectLine(img, centerLeft, centerRight, heavy, 1)
	case 0x2502: // light vertical line
		drawRectLine(img, centerUp, centerDown, light, 1)
	case 0x2503: // heavy vertical line
		drawRectLine(img, centerUp, centerDown, heavy, 1)

	case 0x2504, 0x2505, 0x2506, 0x2507,
		0x2508, 0x2509, 0x250A, 0x250B: // NOTE: Dashes are not supported
		return nil

	// Two lines from center to two direction
	// Every individual line is light or heavy
	case 0x250C, 0x250D, 0x250E, 0x250F,
		0x2510, 0x2511, 0x2512, 0x2513,
		0x2514, 0x2515, 0x2516, 0x2517,
		0x2518, 0x2519, 0x251A, 0x251B:
		n := char - 0x250C
		up := centerDown
		if n/8 > 0 {
			up = centerUp
		}
		left := centerRight
		if (n/4)%2 != 0 {
			left = centerLeft
		}
		upThickness := light
		if (n/2)%2 != 0 {
			upThickness = heavy
		}
		leftThickness := light
		if n%2 != 0 {
			leftThickness = heavy
		}
		drawRectLinesFromPoint(img, 1, center,
			pointWithThickness{vector: up, thickness: upThickness},
			pointWithThickness{vector: left, thickness: leftThickness},
		)

	// Three lines from center to three direction
	// Every individual line is light or heavy
	// There is always a vertical line from top to bottom
	case 0x251C, 0x251D, 0x251E, 0x251F,
		0x2520, 0x2521, 0x2522, 0x2523,
		0x2524, 0x2525, 0x2526, 0x2527,
		0x2528, 0x2529, 0x252A, 0x252B:
		n := char - 0x251C
		right := centerRight
		if n >= 8 {
			right = centerLeft
			n -= 8
		}
		upThickness := light
		if n == 2 || n == 4 || n == 5 || n == 7 {
			upThickness = heavy
		}
		rightThickness := light
		if n == 1 || n == 5 || n == 6 || n == 7 {
			rightThickness = heavy
		}
		downThickness := light
		if n == 3 || n == 4 || n == 6 || n == 7 {
			downThickness = heavy
		}
		drawRectLinesFromPoint(img, 1, center,
			pointWithThickness{vector: centerUp, thickness: upThickness},
			pointWithThickness{vector: right, thickness: rightThickness},
			pointWithThickness{vector: centerDown, thickness: downThickness},
		)

	// Three lines from center to three direction
	// Every individual line is light or heavy
	// There is always a horizontal line from left to right
	case 0x252C, 0x252D, 0x252E, 0x252F,
		0x2530, 0x2531, 0x2532, 0x2533,
		0x2534, 0x2535, 0x2536, 0x2537,
		0x2538, 0x2539, 0x253A, 0x253B:
		n := char - 0x252C
		down := centerDown
		if n >= 8 {
			down = centerUp
		}
		leftThickness := light
		if n%2 != 0 {
			leftThickness = heavy
		}
		rightThickness := light
		if (n/2)%2 != 0 {
			rightThickness = heavy
		}
		downThickness := light
		if (n/4)%2 != 0 {
			downThickness = heavy
		}
		drawRectLinesFromPoint(img, 1, center,
			pointWithThickness{vector: centerLeft, thickness: leftThickness},
			pointWithThickness{vector: down, thickness: downThickness},
			pointWithThickness{vector: centerRight, thickness: rightThickness},
		)

	// Four lines from center to four direction
	// Every individual line is light or heavy
	case 0x253C, 0x253D, 0x253E, 0x253F,
		0x2540, 0x2541, 0x2542, 0x2543,
		0x2544, 0x2545, 0x2546, 0x2547,
		0x2548, 0x2549, 0x254A, 0x254B:
		n := char - 0x253C
		upThickness := light
		if n == 4 || n == 6 || n == 7 || n == 8 || n == 11 || n == 13 || n == 14 || n == 15 {
			upThickness = heavy
		}
		rightThickness := light
		if n == 2 || n == 3 || n == 8 || n == 10 || n == 11 || n == 12 || n == 14 || n == 15 {
			rightThickness = heavy
		}
		downThickness := light
		if n == 5 || n == 6 || n == 9 || n == 10 || n == 12 || n == 13 || n == 14 || n == 15 {
			downThickness = heavy
		}
		leftThickness := light
		if n == 1 || n == 3 || n == 7 || n == 9 || n == 11 || n == 12 || n == 13 || n == 15 {
			leftThickness = heavy
		}
		drawRectLinesFromPoint(img, 1, center,
			pointWithThickness{vector: centerUp, thickness: upThickness},
			pointWithThickness{vector: centerRight, thickness: rightThickness},
			pointWithThickness{vector: centerDown, thickness: downThickness},
			pointWithThickness{vector: centerLeft, thickness: leftThickness},
		)

	case 0x254C, 0x254D, 0x254E, 0x254F: // NOTE: Dashes are not supported
		return nil

	case 0x2550: // double horizontal
		drawDoubleLinesFromPoint(img, centerLeft,
			pointDouble{vector: centerRight, thickness: light, double: true},
		)
	case 0x2551: // double vertical
		drawDoubleLinesFromPoint(img, centerUp,
			pointDouble{vector: centerDown, thickness: light, double: true},
		)
	case 0x2552: // down single and right double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: false},
		)
	case 0x2553: // down double and right single
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: false},
			pointDouble{vector: centerDown, thickness: light, double: true},
		)
	case 0x2554: // down double and right double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: true},
		)
	case 0x2555: // down single and left double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: false},
		)
	case 0x2556: // down double and left single
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerLeft, thickness: light, double: false},
			pointDouble{vector: centerDown, thickness: light, double: true},
		)
	case 0x2557: // down double and left double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: true},
		)
	case 0x2558: // up single and right double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: true},
			pointDouble{vector: centerUp, thickness: light, double: false},
		)
	case 0x2559: // up double and right single
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: false},
			pointDouble{vector: centerUp, thickness: light, double: true},
		)
	case 0x255A: // up double and right double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: true},
			pointDouble{vector: centerUp, thickness: light, double: true},
		)
	case 0x255B: // up single and left double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerUp, thickness: light, double: false},
		)
	case 0x255C: // up double and left single
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerLeft, thickness: light, double: false},
			pointDouble{vector: centerUp, thickness: light, double: true},
		)
	case 0x255D: // up double and left double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerUp, thickness: light, double: true},
		)
	case 0x255E: // vertical single and right double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: false},
			pointDouble{vector: centerUp, thickness: light, double: false},
		)
	case 0x255F: // vertical double and right single
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: false},
			pointDouble{vector: centerDown, thickness: light, double: true},
			pointDouble{vector: centerUp, thickness: light, double: true},
		)
	case 0x2560: // vertical double and right double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerUp, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: true},
			pointDouble{vector: centerRight, thickness: light, double: true},
		)
	case 0x2561: // vertical single and left double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: false},
			pointDouble{vector: centerUp, thickness: light, double: false},
		)
	case 0x2562: // vertical double and left single
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerDown, thickness: light, double: true},
			pointDouble{vector: centerUp, thickness: light, double: true},
			pointDouble{vector: centerLeft, thickness: light, double: false},
		)
	case 0x2563: // vertical double and left double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerDown, thickness: light, double: true},
			pointDouble{vector: centerUp, thickness: light, double: true},
			pointDouble{vector: centerLeft, thickness: light, double: true},
		)
	case 0x2564: // horizontal double and down single
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: true},
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: false},
		)
	case 0x2565: // horizontal single and down double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: false},
			pointDouble{vector: centerLeft, thickness: light, double: false},
			pointDouble{vector: centerDown, thickness: light, double: true},
		)
	case 0x2566: // horizontal double and down double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: true},
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: true},
		)
	case 0x2567: // horizontal double and up single
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: true},
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerUp, thickness: light, double: false},
		)
	case 0x2568: // horizontal single and up double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: false},
			pointDouble{vector: centerLeft, thickness: light, double: false},
			pointDouble{vector: centerUp, thickness: light, double: true},
		)
	case 0x2569: // horizontal double and up double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerRight, thickness: light, double: true},
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerUp, thickness: light, double: true},
		)
	case 0x256A: // vertical single and horizontal double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerRight, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: false},
			pointDouble{vector: centerUp, thickness: light, double: false},
		)
		// drawRectLine(img, centerUp, centerDown, light, 1)
	case 0x256B: // vertical double and horizontal single
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerUp, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: true},
			pointDouble{vector: centerLeft, thickness: light, double: false},
			pointDouble{vector: centerRight, thickness: light, double: false},
		)
		// drawRectLine(img, centerLeft, centerRight, light, 1)
	case 0x256C: // vertical double and horizontal double
		drawDoubleLinesFromPoint(img, center,
			pointDouble{vector: centerUp, thickness: light, double: true},
			pointDouble{vector: centerDown, thickness: light, double: true},
			pointDouble{vector: centerLeft, thickness: light, double: true},
			pointDouble{vector: centerRight, thickness: light, double: true},
		)

	// NOTE: We draw arcs like corners
	case 0x256D: // light down to right arc
		drawRectLinesFromPoint(img, 1, center,
			pointWithThickness{vector: centerDown, thickness: light},
			pointWithThickness{vector: centerRight, thickness: light},
		)
	case 0x256E: // light down to left arc
		drawRectLinesFromPoint(img, 1, center,
			pointWithThickness{vector: centerDown, thickness: light},
			pointWithThickness{vector: centerLeft, thickness: light},
		)
	case 0x256F: // light up to left arc
		drawRectLinesFromPoint(img, 1, center,
			pointWithThickness{vector: centerUp, thickness: light},
			pointWithThickness{vector: centerLeft, thickness: light},
		)
	case 0x2570: // light up to right arc
		drawRectLinesFromPoint(img, 1, center,
			pointWithThickness{vector: centerUp, thickness: light},
			pointWithThickness{vector: centerRight, thickness: light},
		)

	case 0x2571, 0x2572, 0x2573: // NOTE: Diagonals are not supported
		// (?) We can support diagonals by using vector library
		return nil

	// One line from center to one direction
	case 0x2574, 0x2575, 0x2576, 0x2577,
		0x2578, 0x2579, 0x257A, 0x257B:
		n := char - 0x2574
		pos := centerLeft
		switch n % 4 {
		case 1: // up
			pos = centerUp
		case 2: // right
			pos = centerRight
		case 3: // down
			pos = centerDown
		}
		thickness := light
		if n/4 >= 1 {
			thickness = heavy
		}
		drawRectLine(img, center, pos, thickness, 1)

	// Both vertical or horizontal two lines from center to two directions with different thickness
	case 0x257C:
		drawRectLine(img, center, centerLeft, light, 1)
		drawRectLine(img, center, centerRight, heavy, 1)
	case 0x257D:
		drawRectLine(img, center, centerUp, light, 1)
		drawRectLine(img, center, centerDown, heavy, 1)
	case 0x257E:
		drawRectLine(img, center, centerLeft, heavy, 1)
		drawRectLine(img, center, centerRight, light, 1)
	case 0x257F:
		drawRectLine(img, center, centerUp, heavy, 1)
		drawRectLine(img, center, centerDown, light, 1)

	default:
		panic(fmt.Errorf("missing box glyph %d (%s)", char, string(char)))
	}

	return img
}

func (face *Face) DrawUnicodeBlockGlyph(char rune, imgSize common.Vector2[int]) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, imgSize.Width(), imgSize.Height()))
	w := float32(imgSize.Width())
	h := float32(imgSize.Height())

	switch char {
	case 0x2580: // upper 1/2 block
		drawRect(img, common.Rect(0, 0, w, h/2), 1)
	case 0x2581: // lower 1/8 block
		drawRect(img, common.Rect(0, h-h/8, w, h/8), 1)
	case 0x2582: // lower 1/4 block
		drawRect(img, common.Rect(0, h-h/4, w, h/4), 1)
	case 0x2583: // lower 3/8 block
		drawRect(img, common.Rect(0, h-(h*3/8), w, h*3/8), 1)
	case 0x2584: // lower 1/2 block
		drawRect(img, common.Rect(0, h/2, w, h/2), 1)
	case 0x2585: // lower 5/8 block
		drawRect(img, common.Rect(0, h-(h*5/8), w, h*5/8), 1)
	case 0x2586: // lower 3/4 block
		drawRect(img, common.Rect(0, h-(h*3/4), w, h*3/4), 1)
	case 0x2587: // lower 7/8 block
		drawRect(img, common.Rect(0, h-(h*7/8), w, h*7/8), 1)
	case 0x2588: // full block
		drawRect(img, common.Rect(0, 0, w, h), 1)
	case 0x2589: // left 7/8 block
		drawRect(img, common.Rect(0, 0, w*7/8, h), 1)
	case 0x258A: // left 3/4 block
		drawRect(img, common.Rect(0, 0, w*3/4, h), 1)
	case 0x258B: // left 5/8 block
		drawRect(img, common.Rect(0, 0, w*5/8, h), 1)
	case 0x258C: // left 1/2 block
		drawRect(img, common.Rect(0, 0, w/2, h), 1)
	case 0x258D: // left 3/8 block
		drawRect(img, common.Rect(0, 0, w*3/8, h), 1)
	case 0x258E: // left 1/4 block
		drawRect(img, common.Rect(0, 0, w/4, h), 1)
	case 0x258F: // left 1/8 block
		drawRect(img, common.Rect(0, 0, w/8, h), 1)
	case 0x2590: // rigt 1/2 block
		drawRect(img, common.Rect(w/2, 0, w/2, h), 1)
	case 0x2591: // light shade
		drawRect(img, common.Rect(0, 0, w, h), 0.25)
	case 0x2592: // medium shade
		drawRect(img, common.Rect(0, 0, w, h), 0.50)
	case 0x2593: // dark shade
		drawRect(img, common.Rect(0, 0, w, h), 0.75)
	case 0x2594: // upper 1/8 block
		drawRect(img, common.Rect(0, 0, w, h/8), 1)
	case 0x2595: // right 1/8 block
		drawRect(img, common.Rect(w-w/8, 0, w/8, h), 1)
	case 0x2596, 0x2597, 0x2598, 0x2599, 0x259A,
		0x259B, 0x259C, 0x259D, 0x259E, 0x259F: // quadrants
		n := char - 0x2596
		if n >= 2 && n <= 6 { // upper left
			drawRect(img, common.Rect(0, 0, w/2, h/2), 1)
		}
		if n == 0 || n == 3 || n == 5 || n == 8 || n == 9 { // lower left
			drawRect(img, common.Rect(0, h/2, w/2, h/2), 1)
		}
		if n >= 5 && n <= 9 { // upper right
			drawRect(img, common.Rect(w/2, 0, w/2, h/2), 1)
		}
		if n == 1 || n == 3 || n == 4 || n == 6 || n == 9 { // lower right
			drawRect(img, common.Rect(w/2, h/2, w/2, h/2), 1)
		}
	default:
		panic(fmt.Errorf("missing block glyph %d (%s)", char, string(char)))
	}

	return img
}
