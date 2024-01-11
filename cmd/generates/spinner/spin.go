/*
 * Copyright 2023 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package spinner

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"golang.org/x/term"
)

// errInvalidColor is returned when attempting to set an invalid color
var errInvalidColor = errors.New("invalid color")

// validColors holds an array of the only colors allowed
var validColors = map[string]bool{
	// default colors for backwards compatibility
	"black":   true,
	"red":     true,
	"green":   true,
	"yellow":  true,
	"blue":    true,
	"magenta": true,
	"cyan":    true,
	"white":   true,

	// attributes
	"reset":        true,
	"bold":         true,
	"faint":        true,
	"italic":       true,
	"underline":    true,
	"blinkslow":    true,
	"blinkrapid":   true,
	"reversevideo": true,
	"concealed":    true,
	"crossedout":   true,

	// foreground text
	"fgBlack":   true,
	"fgRed":     true,
	"fgGreen":   true,
	"fgYellow":  true,
	"fgBlue":    true,
	"fgMagenta": true,
	"fgCyan":    true,
	"fgWhite":   true,

	// foreground Hi-Intensity text
	"fgHiBlack":   true,
	"fgHiRed":     true,
	"fgHiGreen":   true,
	"fgHiYellow":  true,
	"fgHiBlue":    true,
	"fgHiMagenta": true,
	"fgHiCyan":    true,
	"fgHiWhite":   true,

	// background text
	"bgBlack":   true,
	"bgRed":     true,
	"bgGreen":   true,
	"bgYellow":  true,
	"bgBlue":    true,
	"bgMagenta": true,
	"bgCyan":    true,
	"bgWhite":   true,

	// background Hi-Intensity text
	"bgHiBlack":   true,
	"bgHiRed":     true,
	"bgHiGreen":   true,
	"bgHiYellow":  true,
	"bgHiBlue":    true,
	"bgHiMagenta": true,
	"bgHiCyan":    true,
	"bgHiWhite":   true,
}

var isWindows = runtime.GOOS == "windows"
var isWindowsTerminalOnWindows = len(os.Getenv("WT_SESSION")) > 0 && isWindows

var colorAttributeMap = map[string]color.Attribute{
	"black":        color.FgBlack,
	"red":          color.FgRed,
	"green":        color.FgGreen,
	"yellow":       color.FgYellow,
	"blue":         color.FgBlue,
	"magenta":      color.FgMagenta,
	"cyan":         color.FgCyan,
	"white":        color.FgWhite,
	"reset":        color.Reset,
	"bold":         color.Bold,
	"faint":        color.Faint,
	"italic":       color.Italic,
	"underline":    color.Underline,
	"blinkslow":    color.BlinkSlow,
	"blinkrapid":   color.BlinkRapid,
	"reversevideo": color.ReverseVideo,
	"concealed":    color.Concealed,
	"crossedout":   color.CrossedOut,
	"fgBlack":      color.FgBlack,
	"fgRed":        color.FgRed,
	"fgGreen":      color.FgGreen,
	"fgYellow":     color.FgYellow,
	"fgBlue":       color.FgBlue,
	"fgMagenta":    color.FgMagenta,
	"fgCyan":       color.FgCyan,
	"fgWhite":      color.FgWhite,
	"fgHiBlack":    color.FgHiBlack,
	"fgHiRed":      color.FgHiRed,
	"fgHiGreen":    color.FgHiGreen,
	"fgHiYellow":   color.FgHiYellow,
	"fgHiBlue":     color.FgHiBlue,
	"fgHiMagenta":  color.FgHiMagenta,
	"fgHiCyan":     color.FgHiCyan,
	"fgHiWhite":    color.FgHiWhite,
	"bgBlack":      color.BgBlack,
	"bgRed":        color.BgRed,
	"bgGreen":      color.BgGreen,
	"bgYellow":     color.BgYellow,
	"bgBlue":       color.BgBlue,
	"bgMagenta":    color.BgMagenta,
	"bgCyan":       color.BgCyan,
	"bgWhite":      color.BgWhite,
	"bgHiBlack":    color.BgHiBlack,
	"bgHiRed":      color.BgHiRed,
	"bgHiGreen":    color.BgHiGreen,
	"bgHiYellow":   color.BgHiYellow,
	"bgHiBlue":     color.BgHiBlue,
	"bgHiMagenta":  color.BgHiMagenta,
	"bgHiCyan":     color.BgHiCyan,
	"bgHiWhite":    color.BgHiWhite,
}

func validColor(c string) bool {
	return validColors[c]
}

type Spinner struct {
	mu              *sync.RWMutex
	Delay           time.Duration
	chars           []string
	Prefix          string
	Suffix          string
	FinalMSG        string
	lastOutputPlain string
	LastOutput      string
	color           func(a ...interface{}) string
	Writer          io.Writer
	WriterFile      *os.File
	active          bool
	enabled         bool
	stopChan        chan struct{}
	HideCursor      bool
	PreUpdate       func(s *Spinner)
	PostUpdate      func(s *Spinner)
}

func New(cs []string, d time.Duration, options ...Option) *Spinner {
	s := &Spinner{
		Delay:      d,
		chars:      cs,
		color:      color.New(color.FgWhite).SprintFunc(),
		mu:         &sync.RWMutex{},
		Writer:     color.Output,
		WriterFile: os.Stdout,
		stopChan:   make(chan struct{}, 1),
		active:     false,
		enabled:    true,
		HideCursor: true,
	}

	for _, option := range options {
		option(s)
	}

	return s
}

type Option func(*Spinner)

type Options struct {
	Color      string
	Suffix     string
	FinalMSG   string
	HideCursor bool
}

func WithColor(color string) Option {
	return func(s *Spinner) {
		s.Color(color)
	}
}

func WithSuffix(suffix string) Option {
	return func(s *Spinner) {
		s.Suffix = suffix
	}
}

func WithFinalMSG(finalMsg string) Option {
	return func(s *Spinner) {
		s.FinalMSG = finalMsg
	}
}

func WithHiddenCursor(hideCursor bool) Option {
	return func(s *Spinner) {
		s.HideCursor = hideCursor
	}
}

func WithWriter(w io.Writer) Option {
	return func(s *Spinner) {
		s.mu.Lock()
		s.Writer = w
		s.WriterFile = os.Stdout
		s.mu.Unlock()
	}
}

func (s *Spinner) Active() bool {
	return s.active
}

func (s *Spinner) Enabled() bool {
	return s.enabled
}

func (s *Spinner) Enable() {
	s.enabled = true
	s.Restart()
}

func (s *Spinner) Disable() {
	s.enabled = false
	s.Stop()
}

func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active || !s.enabled || !isRunningInTerminal(s) {
		s.mu.Unlock()
		return
	}
	if s.HideCursor && !isWindowsTerminalOnWindows {
		fmt.Fprint(s.Writer, "\033[?25l")
	}
	if isWindows && !isWindowsTerminalOnWindows {
		color.NoColor = true
	}

	s.active = true
	s.mu.Unlock()

	go func() {
		for {
			for i := 0; i < len(s.chars); i++ {
				select {
				case <-s.stopChan:
					return
				default:
					s.mu.Lock()
					if !s.active {
						s.mu.Unlock()
						return
					}
					if !isWindowsTerminalOnWindows {
						s.erase()
					}

					if s.PreUpdate != nil {
						s.PreUpdate(s)
					}

					var outColor string
					if isWindows {
						if s.Writer == os.Stderr {
							outColor = fmt.Sprintf("\r%s%s%s", s.Prefix, s.chars[i], s.Suffix)
						} else {
							outColor = fmt.Sprintf("\r%s%s%s", s.Prefix, s.color(s.chars[i]), s.Suffix)
						}
					} else {
						outColor = fmt.Sprintf("\r%s%s%s", s.Prefix, s.color(s.chars[i]), s.Suffix)
					}
					outPlain := fmt.Sprintf("\r%s%s%s", s.Prefix, s.chars[i], s.Suffix)
					fmt.Fprint(s.Writer, outColor)
					s.lastOutputPlain = outPlain
					s.LastOutput = outColor
					delay := s.Delay

					if s.PostUpdate != nil {
						s.PostUpdate(s)
					}

					s.mu.Unlock()
					time.Sleep(delay)
				}
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		s.active = false
		if s.HideCursor && !isWindowsTerminalOnWindows {
			// makes the cursor visible
			fmt.Fprint(s.Writer, "\033[?25h")
		}
		s.erase()
		if s.FinalMSG != "" {
			if isWindowsTerminalOnWindows {
				fmt.Fprint(s.Writer, "\r", s.FinalMSG)
			} else {
				fmt.Fprint(s.Writer, s.FinalMSG)
			}
		}
		s.stopChan <- struct{}{}
	}
}

func (s *Spinner) Restart() {
	s.Stop()
	s.Start()
}

func (s *Spinner) Reverse() {
	s.mu.Lock()
	for i, j := 0, len(s.chars)-1; i < j; i, j = i+1, j-1 {
		s.chars[i], s.chars[j] = s.chars[j], s.chars[i]
	}
	s.mu.Unlock()
}

func (s *Spinner) Color(colors ...string) error {
	colorAttributes := make([]color.Attribute, len(colors))

	for index, c := range colors {
		if !validColor(c) {
			return errInvalidColor
		}
		colorAttributes[index] = colorAttributeMap[c]
	}

	s.mu.Lock()
	s.color = color.New(colorAttributes...).SprintFunc()
	s.mu.Unlock()
	return nil
}

func (s *Spinner) UpdateSpeed(d time.Duration) {
	s.mu.Lock()
	s.Delay = d
	s.mu.Unlock()
}

func (s *Spinner) UpdateCharSet(cs []string) {
	s.mu.Lock()
	s.chars = cs
	s.mu.Unlock()
}

func (s *Spinner) erase() {
	n := utf8.RuneCountInString(s.lastOutputPlain)
	if runtime.GOOS == "windows" && !isWindowsTerminalOnWindows {
		clearString := "\r" + strings.Repeat(" ", n) + "\r"
		fmt.Fprint(s.Writer, clearString)
		s.lastOutputPlain = ""
		return
	}

	numberOfLinesToErase := computeNumberOfLinesNeededToPrintString(s.lastOutputPlain)

	eraseCodeString := strings.Builder{}
	eraseCodeString.WriteString("\r\033[K") // start by erasing current line
	for i := 1; i < numberOfLinesToErase; i++ {
		eraseCodeString.WriteString("\033[F\033[K")
	}
	fmt.Fprintf(s.Writer, eraseCodeString.String())
	s.lastOutputPlain = ""
}

func (s *Spinner) Lock() {
	s.mu.Lock()
}

func (s *Spinner) Unlock() {
	s.mu.Unlock()
}

func GenerateNumberSequence(length int) []string {
	numSeq := make([]string, length)
	for i := 0; i < length; i++ {
		numSeq[i] = strconv.Itoa(i)
	}
	return numSeq
}

func isRunningInTerminal(s *Spinner) bool {
	return isatty.IsTerminal(s.WriterFile.Fd())
}

func computeNumberOfLinesNeededToPrintString(linePrinted string) int {
	terminalWidth := math.MaxInt
	if term.IsTerminal(0) {
		if width, _, err := term.GetSize(0); err == nil {
			terminalWidth = width
		}
	}
	return computeNumberOfLinesNeededToPrintStringInternal(linePrinted, terminalWidth)
}

func isAnsiMarker(r rune) bool {
	return r == '\x1b'
}

func isAnsiTerminator(r rune) bool {
	return (r >= 0x40 && r <= 0x5a) || (r == 0x5e) || (r >= 0x60 && r <= 0x7e)
}

func computeLineWidth(line string) int {
	width := 0
	ansi := false

	for _, r := range []rune(line) {
		if ansi || isAnsiMarker(r) {
			ansi = !isAnsiTerminator(r)
		} else {
			width += utf8.RuneLen(r)
		}
	}

	return width
}

func computeNumberOfLinesNeededToPrintStringInternal(linePrinted string, maxLineWidth int) int {
	lineCount := 0
	for _, line := range strings.Split(linePrinted, "\n") {
		lineCount += 1

		lineWidth := computeLineWidth(line)
		if lineWidth > maxLineWidth {
			lineCount += int(float64(lineWidth) / float64(maxLineWidth))
		}
	}

	return lineCount
}
