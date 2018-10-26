package gui

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/tfriedel6/canvas/glfwcanvas"

	"github.com/liamg/aminal/buffer"
	"github.com/liamg/aminal/config"
	"github.com/liamg/aminal/terminal"
	"go.uber.org/zap"
)

type GUI struct {
	logger     *zap.SugaredLogger
	config     *config.Config
	terminal   *terminal.Terminal
	width      int //window width in pixels
	height     int //window height in pixels
	fontScale  float64
	renderer   *OpenGLRenderer
	colourAttr uint32
	mouseDown  bool
	charWidth  float64
	charHeight float64
}

func New(config *config.Config, terminal *terminal.Terminal, logger *zap.SugaredLogger) *GUI {

	return &GUI{
		config:    config,
		logger:    logger,
		width:     800,
		height:    600,
		terminal:  terminal,
		fontScale: 14.0,
	}
}

// inspired by https://kylewbanks.com/blog/tutorial-opengl-with-golang-part-1-hello-opengl

// can only be called on OS thread
func (gui *GUI) resize(w *glfw.Window, width int, height int) {

	gui.logger.Debugf("Initiating GUI resize to %dx%d", width, height)

	gui.width = width
	gui.height = height

	cols := uint(math.Floor(float64(gui.width) / gui.charWidth))
	rows := uint(math.Floor(float64(gui.height) / gui.charHeight))

	gui.logger.Debugf("Resizing internal terminal to %d,%d...", cols, rows)
	if err := gui.terminal.SetSize(cols, rows); err != nil {
		gui.logger.Errorf("Failed to resize terminal to %d cols, %d rows: %s", cols, rows, err)
	}

	gui.logger.Debugf("Resize complete!")

}

func (gui *GUI) glfwScrollCallback(w *glfw.Window, xoff float64, yoff float64) {
	if yoff > 0 {
		gui.terminal.ScrollUp(1)
	} else {
		gui.terminal.ScrollDown(1)
	}
}

func (gui *GUI) getTermSize() (uint, uint) {
	if gui.renderer == nil {
		return 0, 0
	}
	return gui.renderer.GetTermSize()
}

func (gui *GUI) Render() error {

	gui.logger.Debugf("Creating window...")
	wnd, cv, err := glfwcanvas.CreateWindow(gui.width, gui.height, "Aminal")
	if err != nil {
		return fmt.Errorf("Failed to create window: %s", err)
	}
	defer wnd.Close()

	cv.SetFont("./gui/packed-fonts/Hack-Regular.ttf", gui.fontScale)

	titleChan := make(chan bool, 1)

	wnd.Window.SetFramebufferSizeCallback(gui.resize)
	wnd.Window.SetKeyCallback(gui.key)
	wnd.Window.SetCharCallback(gui.char)
	wnd.Window.SetScrollCallback(gui.glfwScrollCallback)
	wnd.Window.SetMouseButtonCallback(gui.mouseButtonCallback)
	wnd.Window.SetCursorPosCallback(gui.mouseMoveCallback)
	wnd.Window.SetRefreshCallback(func(w *glfw.Window) {
		gui.terminal.SetDirty()
	})
	wnd.Window.SetFocusCallback(func(w *glfw.Window, focused bool) {
		if focused {
			gui.terminal.SetDirty()
		}
	})
	w, h := wnd.Window.GetFramebufferSize()
	gui.resize(wnd.Window, w, h)
	cv.SetBounds(0, 0, w, h)

	gui.logger.Debugf("Starting pty read handling...")

	go func() {
		err := gui.terminal.Read()
		if err != nil {
			gui.logger.Errorf("Read from pty failed: %s", err)
		}
		wnd.Close()
		os.Exit(0)
	}()

	gui.logger.Debugf("Starting render...")

	gui.terminal.AttachTitleChangeHandler(titleChan)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	defaultCell := buffer.NewBackgroundCell(gui.config.ColourScheme.Background)

	go func() {
		for {
			select {
			case <-ticker.C:
				gui.logger.Sync()
			case <-titleChan:
				wnd.Window.SetTitle(gui.terminal.GetTitle())
			}
		}
	}()

	textMetrics := cv.MeasureText("█")
	gui.charWidth = textMetrics.Width
	gui.charHeight = textMetrics.ActualBoundingBoxAscent + textMetrics.ActualBoundingBoxDescent

	bg := gui.config.ColourScheme.Background
	cv.SetFillStyle(bg[0], bg[1], bg[2])
	cv.Fill()

	wnd.MainLoop(func() {

		if gui.terminal.CheckDirty() {
			cv.ClearRect(0, 0, float64(gui.width), float64(gui.height))
			lines := gui.terminal.GetVisibleLines()
			lineCount := int(gui.terminal.ActiveBuffer().ViewHeight())
			colCount := int(gui.terminal.ActiveBuffer().ViewWidth())
			for y := 0; y < lineCount; y++ {
				for x := 0; x < colCount; x++ {

					cell := defaultCell
					hasText := false

					if y < len(lines) {
						cells := lines[y].Cells()
						if x < len(cells) {
							cell = cells[x]
							if cell.Rune() != 0 && cell.Rune() != 32 {
								hasText = true
							}
						}
					}

					cursor := false
					if gui.terminal.Modes().ShowCursor {
						cx := uint(gui.terminal.GetLogicalCursorX())
						cy := uint(gui.terminal.GetLogicalCursorY())
						cy = cy + uint(gui.terminal.GetScrollOffset())
						cursor = cx == uint(x) && cy == uint(y)
					}

					var fg [3]float32
					var bg [3]float32

					if cell.Attr().Reverse {
						bg = cell.Fg()
						fg = cell.Bg()
					} else {
						bg = cell.Bg()
						fg = cell.Fg()
					}

					if cursor {
						bg = gui.config.ColourScheme.Cursor
					} else if gui.terminal.ActiveBuffer().InSelection(uint16(x), uint16(y)) {
						bg = gui.config.ColourScheme.Selection
					}

					px := (float64(x) * textMetrics.Width)
					py := math.Floor(float64(y) * gui.charHeight)

					cv.SetFillStyle(bg[0], bg[1], bg[2])
					cv.FillRect(px, py, textMetrics.Width, gui.charHeight)
					if hasText {
						cv.SetFillStyle(fg[0], fg[1], fg[2])
						cv.FillText(string(cell.Rune()), (px), (py + textMetrics.ActualBoundingBoxAscent))
						//gui.logger.Errorf("Co-ords", px, py)
					}
				}
			}
		}

	})

	gui.logger.Debugf("Stopping render...")
	return nil

}
