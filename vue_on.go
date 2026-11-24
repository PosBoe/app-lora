//go:build vue

package lora

import (
	"embed"
	"io/fs"
)

//go:embed vue/dist
var vueDistFS embed.FS

// VueDistFS returns the built frontend. The vue build tag embeds it; run
// `task build:vue` first.
func VueDistFS() fs.FS {
	fsys, err := fs.Sub(vueDistFS, "vue/dist")
	if err != nil {
		panic(err)
	}
	return fsys
}
