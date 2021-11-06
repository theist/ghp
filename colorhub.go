package main

import (
	"math"
	"math/rand"

	jump "github.com/lithammer/go-jump-consistent-hash"

	"github.com/gookit/color"
)

type colorHub struct {
	colors map[string]color.RGBColor
	hasher *jump.Hasher
}

// main colorhub, will be global to program
var singleColorHub colorHub

func (c *colorHub) hashedColor(seed string) color.RGBColor {
	generator := rand.New(rand.NewSource(int64(c.hasher.Hash(seed))))
	r := uint8(generator.Float32() * 255)
	g := uint8(generator.Float32() * 255)
	b := uint8(generator.Float32() * 255)
	return color.RGB(r, g, b)
}

func (c *colorHub) init() {
	c.hasher = jump.New(math.MaxInt32, jump.NewCRC64())
	c.colors = make(map[string]color.RGBColor)
}

func (c *colorHub) stableColorize(str string) string {
	col, exists := c.colors[str]
	if !exists {
		col = c.hashedColor(str)
		c.colors[str] = col
	}
	return col.Sprintf("%v", str)
}

func (c *colorHub) keyColorize(key, str string) string {
	col, exists := c.colors[key]
	if !exists {
		return str
	}
	h := col.Sprintf("foo %v", str)
	return h
}
