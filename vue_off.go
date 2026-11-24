//go:build !vue

package lora

import "io/fs"

// VueDistFS returns nil without the vue build tag. In development, the Vite
// dev server serves the frontend and proxies /api to the Go server.
func VueDistFS() fs.FS {
	return nil
}
