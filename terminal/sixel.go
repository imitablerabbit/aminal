package terminal

import (
	"fmt"

	"github.com/liamg/aminal/sixel"
)

func sixelHandler(pty chan rune, terminal *Terminal) error {

	data := []rune{}

	for {
		b := <-pty
		if b == 0x1b { // terminated by ESC bell or ESC \
			_ = <-pty // swallow \ or bell
			break
		}
		data = append(data, b)
	}

	six, err := sixel.ParseString(string(data))
	if err != nil {
		return fmt.Errorf("Failed to parse sixel data: %s", err)
	}

	x, y := terminal.ActiveBuffer().CursorColumn(), terminal.ActiveBuffer().CursorLine()
	terminal.ActiveBuffer().Write(' ')
	cell := terminal.ActiveBuffer().GetCell(x, y)
	if cell == nil {
		return fmt.Errorf("Missing cell for sixel")
	}

	panic("Not implemented")
	_ = six

	return nil
}
