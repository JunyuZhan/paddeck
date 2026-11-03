package web

import "embed"

//go:embed index.html app.js themes.js style.css theme-*.css
var FS embed.FS
