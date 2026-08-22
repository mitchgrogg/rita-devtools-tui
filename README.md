# rita-devtools-tui

A terminal UI for controlling the rita-mitm REST API - an addon for mitmproxy which runs in [rita-devtools](https://github.com/mitchgrogg/rita-devtools). Manage request delays and request/response alterations from the comfort of your terminal.

## Installation

### From releases

Download the latest binary from the [releases page](https://github.com/mitchgrogg/rita-devtools-tui/releases).

### From source

```bash
go install github.com/mitchgrogg/rita-devtools-tui@latest
```

## Usage

```bash
# Pass the API URL directly
rita-devtools-tui --api http://192.168.86.171:8082

# Or use an environment variable
export RITA_MITM_URL=http://192.168.86.171:8082
rita-devtools-tui

# If neither is set, you'll be prompted to enter it
rita-devtools-tui

# Print version
rita-devtools-tui --version
```

**API URL priority:** `--api` flag > `RITA_MITM_URL` env var > interactive prompt

## Keybindings

### Navigation

| Key                 | Action      |
| ------------------- | ----------- |
| `tab` / `shift+tab` | Switch tabs |
| `1` - `4`           | Jump to tab |
| `↑` / `k`           | Move up     |
| `↓` / `j`           | Move down   |
| `q` / `ctrl+c`      | Quit        |

### Delays Tab

| Key | Action                  |
| --- | ----------------------- |
| `g` | Edit global delay       |
| `a` | Add pattern delay       |
| `d` | Delete selected pattern |
| `D` | Delete all patterns     |
| `r` | Refresh                 |

### Request Alterations Tab

Rewrites outgoing requests before they leave the proxy. Each rule matches a URL
pattern and sets a rewrite URL (a regex substitution applied to the matched part
of the URL, so path and query survive), a replacement request body, or both.

Anchor the pattern with `^` when the rewrite URL is a full URL — e.g. pattern
`^https://api\.prod\.com` with rewrite URL `https://api.staging.com`. An
unanchored pattern would splice the replacement into the middle of the URL;
rita-mitm skips such a rewrite and logs a warning instead of applying it.

| Key | Action                     |
| --- | -------------------------- |
| `a` | Add alteration             |
| `d` | Delete selected alteration |
| `D` | Delete all alterations     |
| `r` | Refresh                    |

### Response Alterations Tab

Overwrites the status code and/or body of responses whose URL matches the rule.

| Key | Action                     |
| --- | -------------------------- |
| `a` | Add alteration             |
| `d` | Delete selected alteration |
| `D` | Delete all alterations     |
| `r` | Refresh                    |

### Settings Tab

| Key       | Action                        |
| --------- | ----------------------------- |
| `enter`   | Select action (export/import) |
| `↑` / `k` | Move up                       |
| `↓` / `j` | Move down                     |

### Forms

| Key     | Action     |
| ------- | ---------- |
| `tab`   | Next field |
| `enter` | Submit     |
| `esc`   | Cancel     |

## Building from source

```bash
git clone https://github.com/mitchgrogg/rita-devtools-tui.git
cd rita-devtools-tui
go build -o rita-devtools-tui .
```
