//go:build !with_labs

package ari

import goal "codeberg.org/anaseto/goal"

func importLabs(_ *goal.Context) error { return nil }
