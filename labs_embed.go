//go:build with_labs

package ari

import (
	"embed"
	"io/fs"

	goal "codeberg.org/anaseto/goal"
	goalfs "codeberg.org/anaseto/goal/io/fs"
)

//go:embed labs
var labsFS embed.FS

func importLabs(ctx *goal.Context) error {
	sub, err := fs.Sub(labsFS, "labs")
	if err != nil {
		return err
	}
	ctx.AssignGlobal("labs", goalfs.NewFS(sub, "labs"))
	return nil
}
