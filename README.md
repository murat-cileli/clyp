# Clyp

Modern, fast, simple clipboard manager for Linux.

<img src="https://raw.githubusercontent.com/murat-cileli/clyp/refs/heads/master/screenshot-1.png" style="max-width:820px;">

## Key Features

- **Native application** written in Go and GTK4.
- **Modern, clean, simple interface** with minimal distractions.
- **Keyboard centric** - Navigate, search, copy and delete items with keyboard.
- **High performance** - Optimized SQLite backend tested with 10,000+ records.
- **Supports text and image content** (up to 3 images) with image previews.
- **Native Wayland and X11 clipboard monitoring** without loading GTK in the background watcher.

## Installation

### Debian 13 / Ubuntu 24.04 and Later
- Download [DEB Package](https://github.com/murat-cileli/clyp/releases/download/0.9.6/clyp_0.9.6_amd64.deb)
- `apt install ./clyp_0.9.6_amd64.deb`

> [!IMPORTANT]
> Depends on `libgtk-4-bin`

### Arch Linux / Manjaro
- Download [Arch Linux Package](https://github.com/murat-cileli/clyp/releases/download/0.9.6/clyp-0.9.6-1-x86_64.pkg.tar.zst)  
- `pacman -U clyp-0.9.6-1-x86_64.pkg.tar.zst`

> [!IMPORTANT]
> Depends on `gtk4`

## Usage

### Starting the Application
```bash
clyp
```

Or launch from your application menu.

### Starting the Watcher at Login

Configure `/usr/bin/clyp-watcher` to run at login. This is the lightweight,
interface-free process responsible for monitoring the clipboard on X11 and
Wayland. The `clyp` executable should only be started when you want to open the
GTK interface and select an item from the history.

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+F` | Toggle search |
| `a` - `z` | Type to search |
| `Enter` | (On search bar) Focus first item on the list |
| `Enter` or `Double Click` | (On the list) Copy selected item to clipboard |
| `Delete` | Remove selected item |
| `Escape` | Hide search bar / Close window |
| `↑/↓` | Navigate through clipboard history |

### Basic Operations

1. **Automatic Clipboard Monitoring**: Clyp automatically captures text and images copied to your clipboard
2. **Browse History**: Use the main window to browse through your clipboard history
3. **Search**: Press `Ctrl+F` to search through your clipboard content
4. **Type to Search**: Start typing to search history instantly
5. **Quick Copy**: Select any item and press `Enter` to copy it back to your clipboard
6. **Delete Items**: Select unwanted items and press `Delete` to remove them

## Technical Details

<img src="https://raw.githubusercontent.com/murat-cileli/clyp/refs/heads/master/architecture-1.png?v=2" style="max-width:622px;">

The application is split into two executables. `clyp` contains the GTK interface,
while `clyp-watcher` is a lightweight process that talks directly to X11/XFixes
or Wayland `ext-data-control-v1` and notifies the GUI of database changes through
a UNIX socket. On Wayland compositors without `ext-data-control-v1`, it falls
back to XWayland when available.

### Architecture
- **Language**: Go 1.25.0
- **GUI Framework**: GTK4 via gotk4 bindings
- **Database**: SQLite3 for persistent storage
- **Platform**: Linux (Wayland/X11)
- **Data Directory**: `~/.local/share/bio.murat.clyp/`
- **SQLite3 Database File**: `~/.local/share/bio.murat.clyp/clyp.db`

## Building from Source (Ubuntu)

```bash
git clone https://github.com/murat-cileli/clyp.git
cd clyp
go mod download
sudo apt install -y \
  libglib2.0-dev \
  libgirepository1.0-dev \
  libgraphene-1.0-dev \
  libcairo2-dev \
  pkg-config \
  build-essential \
  libpango1.0-dev \
  libgdk-pixbuf-2.0-dev \
  libgtk-4-dev \
  libwayland-dev \
  libx11-dev \
  libxfixes-dev
go build -o clyp .
go build -o clyp-watcher ./cmd/clyp-watcher
```

### TODO
- Add import/export.
- Add database encryption.

### Contributing

Contributions are welcome! Please **open an Issue first** before submitting any code changes. Unsolicited Pull Requests will not be accepted. This process helps us discuss your ideas and ensure they align with the project goals before implementation.

### Contributors

<a href="https://github.com/murat-cileli/clyp/graphs/contributors" target="_blank">
  <img src="https://contrib.rocks/image?repo=murat-cileli/clyp" />
</a>

### CREDITS
- [gotk4](https://github.com/diamondburned/gotk4)
- [go-sqlite3](https://github.com/mattn/go-sqlite3)
- [GoReleaser](https://goreleaser.com/)
- [nFPM](https://nfpm.goreleaser.com/)
- [Icon by Freepik - Flaticon](https://www.flaticon.com/free-icons/clipboard)
