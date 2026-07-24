package main

import (
	"fmt"
	"strings"
)

type Progress struct {
	total    int
	current  int
	success  int
	errors   int
	barWidth int
}

func NewProgress(total int) *Progress {
	return &Progress{
		total:    total,
		barWidth: 40,
	}
}

func (p *Progress) Success() {
	p.current++
	p.success++
}

func (p *Progress) Error() {
	p.current++
	p.errors++
}

func (p *Progress) Render() {
	if p.total == 0 {
		return
	}

	percent := p.current * 100 / p.total

	filled := p.current * p.barWidth / p.total
	empty := p.barWidth - filled

	bar := strings.Repeat("█", filled) +
		strings.Repeat("░", empty)

	fmt.Printf(
		"\r%s %3d%%  %s",
		bar,
		percent,
		TF("progress_render", p.current, p.total),
	)
}

func (p *Progress) Finish() {
	fmt.Println()
}
