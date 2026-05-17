---
title: theme.jsonc
description: Reference for ~/.config/jump/theme.jsonc — terminal color palette.
tableOfContents:
  maxHeadingLevel: 3
---

<!-- Generated from apps/jump-web/src/settings-schema.ts — edit the schema, then run pnpm generate. -->

:::note
This page is generated from the [validation schema](https://github.com/sting8k/jump/blob/main/apps/jump-web/src/settings-schema.ts).
:::

`~/.config/jump/theme.jsonc` (or `$XDG_CONFIG_HOME/jump/theme.jsonc`)

Terminal color palette. All fields are optional CSS color strings.
Omitted colors use the built-in defaults shown below.

This file is drop-in compatible with [Windows Terminal themes](https://github.com/mbadolato/iTerm2-Color-Schemes/tree/master/windowsterminal):
`purple`/`brightPurple` are mapped to `magenta`/`brightMagenta`, and the `name` field is ignored.

## Example

```jsonc
{
  "background": "#282a36",
  "foreground": "#f8f8f2",
  "cursor": "#f8f8f2",
  "selectionBackground": "#44475a",
  "black": "#21222c",
  "red": "#ff5555",
  "green": "#50fa7b",
  "yellow": "#f1fa8c",
  "blue": "#bd93f9",
  "purple": "#ff79c6",   // mapped to magenta
  "cyan": "#8be9fd",
  "white": "#f8f8f2"
}
```

## Fields

### `foreground`

Default text color.

- **Default:** `#cdd6f4`

### `background`

Terminal background color.

- **Default:** `#11111b`

### `cursor`

Cursor color.

- **Default:** `#a6e3a1`

### `cursorAccent`

Cursor accent color (text under block cursor).

- **Default:** `#11111b`

### `selectionBackground`

Selection highlight color.

- **Default:** `#313244cc`

### `selectionForeground`

Text color inside selection.


### `selectionInactiveBackground`

Selection color when terminal is not focused.


### `black`

ANSI black.

- **Default:** `#181825`

### `red`

ANSI red.

- **Default:** `#f38ba8`

### `green`

ANSI green.

- **Default:** `#a6e3a1`

### `yellow`

ANSI yellow.

- **Default:** `#f9e2af`

### `blue`

ANSI blue.

- **Default:** `#89b4fa`

### `magenta`

ANSI magenta.

- **Default:** `#cba6f7`

### `cyan`

ANSI cyan.

- **Default:** `#94e2d5`

### `white`

ANSI white.

- **Default:** `#cdd6f4`

### `brightBlack`

ANSI bright black.

- **Default:** `#45475a`

### `brightRed`

ANSI bright red.

- **Default:** `#eba0ac`

### `brightGreen`

ANSI bright green.

- **Default:** `#a6e3a1`

### `brightYellow`

ANSI bright yellow.

- **Default:** `#f9e2af`

### `brightBlue`

ANSI bright blue.

- **Default:** `#89b4fa`

### `brightMagenta`

ANSI bright magenta.

- **Default:** `#cba6f7`

### `brightCyan`

ANSI bright cyan.

- **Default:** `#94e2d5`

### `brightWhite`

ANSI bright white.

- **Default:** `#f5e0dc`

### `purple`

Alias for `magenta` (Windows Terminal compat).


### `brightPurple`

Alias for `brightMagenta` (Windows Terminal compat).


### `name`

Theme name (ignored, present in Windows Terminal theme files).

