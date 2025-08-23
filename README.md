# Clyp

Modern, fast, simple clipboard manager for Linux.

<img src="https://raw.githubusercontent.com/murat-cileli/clyp/refs/heads/master/screenshot-1.png" style="max-width:820px;">

## Key Features

- **Native application** written in Go and GTK4.
- **Modern, clean, simple interface** with minimal distractions.
- **Keyboard centric** - Navigate, search, copy and delete items with keyboard.
- **High performance** - Optimized SQLite backend tested with 10,000+ records.
- **Supports text and image content** (up to 3 images) with image previews.
- **Full Wayland support** - Works natively on both Wayland and X11.

## Installation

### Debian / Ubuntu and Derivatives
- Download [DEB Package](https://github.com/murat-cileli/clyp/releases/download/0.9.2/clyp_0.9.2_UBUNTU_amd64.deb)
- `apt install ./clyp_0.9.2_UBUNTU_amd64.deb`

> [!IMPORTANT]
> Depends on `libgtk-4-dev`

### Arch Linux and Derivatives
- Download [Arch Linux Package](https://github.com/murat-cileli/clyp/releases/download/0.9.2/clyp-0.9.2-1-ARCHLINUX-x86_64.pkg.tar.zst)  
- `pacman -U clyp-0.9.2-1-ARCHLINUX-x86_64.pkg.tar.zst`

> [!IMPORTANT]
> Depends on `gtk4`

## Usage

### Starting the Application
```bash
clyp
```

Or launch from your application menu.

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+F` | Toggle search |
| `Enter` | Copy selected item to clipboard |
| `Delete` | Remove selected item |
| `Escape` | Clear search / Close search bar |
| `↑/↓` | Navigate through clipboard history |

### Basic Operations

1. **Automatic Clipboard Monitoring**: Clyp automatically captures text and images copied to your clipboard
2. **Browse History**: Use the main window to browse through your clipboard history
3. **Search**: Press `Ctrl+F` to search through your clipboard content
4. **Quick Copy**: Select any item and press `Enter` to copy it back to your clipboard
5. **Delete Items**: Select unwanted items and press `Delete` to remove them

## Technical Details

<img src="https://raw.githubusercontent.com/murat-cileli/clyp/refs/heads/master/architecture-1.png?v=2" style="max-width:622px;">

The watcher is a minimal headless Gtk application. It monitors the clipboard and notifies the GUI of database changes via a UNIX socket.

### Architecture
- **Language**: Go 1.25.0
- **GUI Framework**: GTK4 via gotk4 bindings
- **Database**: SQLite3 for persistent storage
- **Platform**: Linux (Wayland/X11)

### Data Storage
Clipboard data is stored in `~/.local/share/bio.murat.clyp/clyp.db` using SQLite3. The database includes:
- Automatic timestamps for each clipboard entry
- Content type detection (text/image)
- Duplicate prevention
- Efficient indexing for fast searches

## Configuration

Clyp follows XDG Base Directory specifications:
- **Data Directory**: `~/.local/share/bio.murat.clyp/`
- **Database File**: `~/.local/share/bio.murat.clyp/clyp.db`

## Development

### Building from Source

#### Ubuntu 

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
  libgtk-4-dev
go build .
```

#### Others

```bash
git clone https://github.com/murat-cileli/clyp.git
cd clyp
go mod download
go build .
```

### Dependencies
- `github.com/diamondburned/gotk4/pkg` - GTK4 bindings for Go
- `github.com/mattn/go-sqlite3` - SQLite3 driver for Go

### Project Structure
```
├── app.go          # Main application logic and UI setup
├── clipboard.go    # Clipboard monitoring and operations
├── database.go     # SQLite database operations
├── main.go         # Application entry point
├── resources/      # UI definitions and CSS styles
├── data/           # Desktop files and metadata
└── vendor/         # Vendored dependencies
```

### TODO
- Add import/export.
- Add database encryption.

### CREDITS
- [gotk4](https://github.com/diamondburned/gotk4)
- [go-sqlite3](https://github.com/mattn/go-sqlite3)
- [GoReleaser](https://goreleaser.com/)
- [nFPM](https://nfpm.goreleaser.com/)
- [Icon by Freepik - Flaticon](https://www.flaticon.com/free-icons/clipboard)
