package main

import (
	ui "github.com/gizak/termui"
)

type pBar struct{
	bar *ui.Gauge
	num int
}

func newBar(num int) *pBar{
	err := ui.Init()
	if err != nil {
		panic(err)
	}

	bar := ui.NewGauge()
	bar.Percent = 0
	bar.Width = 100
	bar.Height = 3
	bar.BorderLabel = "Progress"
	bar.BarColor = ui.ColorGreen
	bar.BorderFg = ui.ColorWhite
	bar.BorderLabelFg = ui.ColorCyan

	ui.Render(bar)

	return &pBar{
		bar: bar,
		num: num,
	}
}

func (b *pBar) set(percent int) {
	b.bar.Percent = percent
	ui.Render(b.bar)
}
