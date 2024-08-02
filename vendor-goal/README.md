# Vendor Goal Code

Goal code that is loaded by default into the Ari runtime is located here as files, embedded into the Ari Go program using `//go:embed` directives. The sections below describe where the code has been taken from.

## This repo

- [shape.goal](shape.goal)
  - Implementation of `reshape` by discodoug, shared [on Matrix](https://matrix.to/#/!laJBzNwLcAOMAbAEeQ:matrix.org/$VnF4KPl4GZKc7F0kYXQ3nUq_4mQJaUIiXNrg0ziHU08?via=matrix.org&via=t2bot.io&via=matrix.fedibird.com)
  - Implementation of `shape` by John Earnest, shared [in the k tree StackExchange chat](https://chat.stackexchange.com/transcript/message/54070438#54070438)

## [goal] repo

Last fetched at commit `a7797629ad427b77f3608f1c3a0c725e5cb692d3`:

```
commit a7797629ad427b77f3608f1c3a0c725e5cb692d3
Author: Yon <anaseto@bardinflor.perso.aquilenet.fr>
Date:   Mon Jul 15 15:53:23 2024 +0000

    add a couple of examples
```

The Goal code vendored in this repository is licensed as follows:

> Copyright (c) 2022 Yon <anaseto@bardinflor.perso.aquilenet.fr>
>
> Permission to use, copy, modify, and distribute this software for any
> purpose with or without fee is hereby granted, provided that the above
> copyright notice and this permission notice appear in all copies.
>
> THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
> WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
> MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
> ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
> WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
> ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
> OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

- [fmt.goal](fmt.goal)
- [html.goal](html.goal)
- [k.goal](k.goal)
- [math.goal](math.goal)
- [mods.goal](mods.goal)

<!-- Links -->

[goal]: https://codeberg.org/anaseto/goal
