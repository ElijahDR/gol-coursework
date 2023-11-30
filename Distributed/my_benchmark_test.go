package main

import (
	"fmt"
	"os"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

const benchLength = 1000

func BenchmarkGol(b *testing.B) {
	for threads := 1; threads <= 16; threads++ {
		os.Stdout = nil // Disable all program output apart from benchmark results
		p := gol.Params{
			Turns:       100,
			Threads:     threads,
			ImageWidth:  512,
			ImageHeight: 512,
		}
		name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				events := make(chan gol.Event)
				go gol.Run(p, events, nil)
				for range events {

				}
			}
		})
	}
}

func BenchmarkGolBig(b *testing.B) {
	for threads := 4; threads <= 16; threads += 4 {
		os.Stdout = nil // Disable all program output apart from benchmark results
		p := gol.Params{
			Turns:       100,
			Threads:     threads,
			ImageWidth:  5120,
			ImageHeight: 5120,
		}
		name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				events := make(chan gol.Event)
				go gol.Run(p, events, nil)
				for range events {

				}
			}
		})
	}
}

func BenchmarkServer(b *testing.B) {
	os.Stdout = nil // Disable all program output apart from benchmark results
	p := gol.Params{
		Turns:       100,
		Threads:     1,
		ImageWidth:  512,
		ImageHeight: 512,
	}
	name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
	b.Run(name, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			events := make(chan gol.Event)
			go gol.Run(p, events, nil)
			for range events {

			}
		}
	})
}

func BenchmarkGolThreads(b *testing.B) {
	os.Stdout = nil // Disable all program output apart from benchmark results
	p := gol.Params{
		Turns:       100,
		Threads:     14,
		ImageWidth:  512,
		ImageHeight: 512,
	}
	name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
	b.Run(name, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			events := make(chan gol.Event)
			go gol.Run(p, events, nil)
			for range events {

			}
		}
	})
}
