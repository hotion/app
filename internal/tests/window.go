package tests

import (
	"testing"

	"github.com/murlokswarm/app"
	"github.com/stretchr/testify/assert"
)

func testWindow(t *testing.T, w app.Window) {
	// app.Elem
	called := false
	w.WhenView(func(n app.View) {
		called = true
	})
	assert.True(t, called)

	called = false
	w.WhenWindow(func(w app.Window) {
		called = true
	})
	assert.True(t, called)

	called = false
	w.WhenMenu(func(m app.Menu) {
		called = true
	})
	assert.False(t, called)

	called = false
	w.WhenDockTile(func(d app.DockTile) {
		called = true
	})
	assert.False(t, called)

	called = false
	w.WhenStatusMenu(func(s app.StatusMenu) {
		called = true
	})
	assert.False(t, called)

	t.Run("view navigation", func(t *testing.T) {
		testViewNav(t, w, false)
	})

	t.Run("compo", func(t *testing.T) {
		testView(t, w)
	})

	w.Position()
	assertElem(t, w)

	w.Move(42, 42)
	assertElem(t, w)

	w.Center()
	assertElem(t, w)

	w.Size()
	assertElem(t, w)

	w.Resize(42, 42)
	assertElem(t, w)

	w.Focus()
	assertElem(t, w)

	w.FullScreen()
	assertElem(t, w)

	w.ExitFullScreen()
	assertElem(t, w)

	w.Minimize()
	assertElem(t, w)

	w.Deminimize()
	assertElem(t, w)

	w.Close()
	assertElem(t, w)
}
