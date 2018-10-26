package sixel

import (
	"fmt"
	"strconv"
	"strings"
)

type Sixel struct {
	rows []row
}

type colour [3]uint8
type column [6]colour
type row []column

func decompress(data string) string {

	output := ""

	inMarker := false
	countStr := ""

	for r := range data {

		if !inMarker {
			if r == '!' {
				inMarker = true
				countStr = ""
			} else {
				output = fmt.Sprintf("%s%c", output, r)
			}
			continue
		}

		if r >= 0x30 && r <= 0x39 {
			countStr = fmt.Sprintf("%s%c", countStr, r)
		} else {
			count, _ := strconv.Atoi(countStr)
			for i := 0; i < count; i++ {
				output = fmt.Sprintf("%s%c", output, r)
			}
			inMarker = false
		}
	}

	return data
}

// pass in everything after ESC+P and before ST
func ParseString(data string) (*Sixel, error) {

	data = decompress(data)

	inHeader := true
	inColour := false

	six := Sixel{}
	var x, y uint

	colourStr := ""

	colourMap := map[string]colour{}
	var selectedColour colour

	// read p1 p2 p3
	for i, r := range data {
		switch true {
		case inHeader:
			// todo read p1 p2 p3
			if r == 'q' {
				inHeader = false
			}
		case inColour:
			colourStr = fmt.Sprintf("%s%c", colourStr, r)
			if i+1 >= len(data) || data[i+1] < 0x30 || data[i+1] > 0x3b {
				// process colour string
				inColour = false
				parts := strings.Split(colourStr, ";")

				// select colour
				if len(parts) == 1 {
					c, ok := colourMap[parts[0]]
					if ok {
						selectedColour = c
					}
				} else if len(parts) == 5 {
					switch parts[1] {
					case "1":
						// HSL
						return nil, fmt.Errorf("HSL colours are not yet supported")
					case "2":
						// RGB
						r, _ := strconv.Atoi(parts[2])
						g, _ := strconv.Atoi(parts[3])
						b, _ := strconv.Atoi(parts[4])
						colourMap[parts[0]] = colour([3]uint8{
							uint8(r & 0xff),
							uint8(g & 0xff),
							uint8(b & 0xff),
						})
					default:
						return nil, fmt.Errorf("Unknown colour definition type: %s", parts[1])
					}
				} else {
					return nil, fmt.Errorf("Invalid colour directive: #%s", colourStr)
				}

				colourStr = ""
			}

		default:
			switch r {
			case '-':
				y += 6
				x = 0
			case '$':
				x = 0
			case '#':
				inColour = true
			default:
				b := (r & 0xff) - 0x3f
				var bit int
				for bit = 5; bit >= 0; bit-- {
					if b&(1<<uint(bit)) > 0 {
						six.setPixel(x, y+uint(bit), selectedColour)
					}
				}
				x++
			}
		}
	}

	return &six, nil
}

func (six *Sixel) setPixel(x, y uint, c colour) {

	if six.rows == nil {
		six.rows = []row{}
	}

	rowY := (y - (y % 6)) / 6

	for rowY >= uint(len(six.rows)) {
		six.rows = append(six.rows, row{})
	}

	panic("Not implemented")

}
