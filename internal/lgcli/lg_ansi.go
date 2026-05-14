//go:build !plan9

/*
 * Copyright (c) 2021 Marcin Gasperowicz <xnooga@gmail.com>
 * SPDX-License-Identifier: MIT
 */

package lgcli

const (
	ansiBold     = "[1m"
	ansiBoldCyan = "[1;36m"
	ansiDim      = "[90m"
	ansiReset    = "[0m"

	bannerQuitHint = "Ctrl-C to quit"
)
