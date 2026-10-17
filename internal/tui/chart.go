package tui

import (
	"fmt"
	"math"
	"strings"
)

// barChart draws values as an ASCII column chart h rows high and at most w
// wide, newest value at the right. The left gutter labels the top and
// bottom of the scale.
func barChart(values []float64, w, h int, top float64, label func(float64) string) []string {
	const gutter = 8
	cols := w - gutter - 1
	if cols < 1 || h < 1 || top <= 0 {
		return nil
	}
	if len(values) > cols {
		values = values[len(values)-cols:]
	}
	lines := make([]string, h)
	for r := range h {
		var b strings.Builder
		for _, v := range values {
			cells := int(math.Round(v / top * float64(h)))
			if v > 0 && cells == 0 {
				cells = 1 // anything above zero gets one cell
			}
			if h-r <= cells {
				b.WriteByte('#')
			} else {
				b.WriteByte(' ')
			}
		}
		axis := ""
		switch r {
		case 0:
			axis = label(top)
		case h - 1:
			axis = label(0)
		}
		lines[r] = fmt.Sprintf("%*s |", gutter-2, axis) + barStyle.Render(b.String())
	}
	return lines
}

// scaleCPU rounds the top of a CPU chart up to a tidy value (1, 2, 2.5 or
// 5 times a power of ten) so small loads stay visible. A process tree can
// use more than one core, so the value may exceed 100%.
func scaleCPU(peak float64) float64 {
	peak = math.Max(peak, 1)
	pow := math.Pow(10, math.Floor(math.Log10(peak)))
	for _, step := range []float64{1, 2, 2.5, 5, 10} {
		if top := step * pow; top >= peak {
			return top
		}
	}
	return 10 * pow
}

// scaleMem leaves headroom above the peak.
func scaleMem(peak float64) float64 {
	return math.Max(peak*1.2, 1<<20)
}

func stats(values []float64) (now, avg, peak float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
		peak = math.Max(peak, v)
	}
	return values[len(values)-1], sum / float64(len(values)), peak
}
