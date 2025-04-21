package main

import (
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var progress *mpb.Progress

func InitProgress() {
	progress = mpb.New(mpb.WithWidth(64))
}

func WaitForProgress() {
	progress.Wait()
}

func NewDownloadBar(title string, total int64) *mpb.Bar {
	return progress.New(
		total,
		mpb.BarStyle().Rbound("|").Lbound("|").Filler("█"),
		mpb.PrependDecorators(
			decor.Name(title),
			decor.Percentage(decor.WC{W: 4}),
		),
		mpb.AppendDecorators(
			decor.Elapsed(decor.ET_STYLE_GO),
		),
	)
}
