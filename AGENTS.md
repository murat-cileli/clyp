# AGENTS.md

## Scope and instruction precedence

This file applies to the entire repository. If a more deeply nested `AGENTS.md`
is added later, its instructions take precedence for files in that subtree.

Use the repository itself as the source of truth for current behavior. Use
authoritative web sources when an important external fact may have changed
(for example, GTK, Go, Flatpak, GoReleaser, nFPM, distribution, or store
requirements); do not guess current facts from memory. Keep research focused
on the task and prefer primary documentation.

## Project overview

Clyp is a Linux clipboard-history application. It is a single Go `main`
package and a single executable with two runtime modes:

- `clyp` starts the GTK4 history browser.
- `clyp --watch` starts a minimal GTK clipboard watcher.

The GUI starts the watcher as a child command. The watcher writes clipboard
items to SQLite, then sends a one-byte notification to the GUI over a Unix
socket. The GUI reloads the visible rows from SQLite after a notification.
The watcher can remain useful when the GUI is closed; failure to reach the GUI
socket is therefore logged but is not fatal.

Core project identifiers and constraints:

- Go module: `github.com/murat-cileli/clyp`
- GTK application ID: `bio.murat.clyp`
- License: GPL-3.0-only
- Primary platform: Linux
- Release packaging currently targets Linux `amd64`
- GTK is accessed through gotk4; SQLite is accessed through
  `github.com/mattn/go-sqlite3`
- Both dependencies use CGO, so a C toolchain and GTK development packages are
  required even for compile-only tests
- The repository vendors its Go dependencies, and project build definitions
  use `-mod=vendor`

This is a privacy-sensitive application. Clipboard text and PNG image data are
stored locally in plaintext. Never print, upload, commit, or include a user's
real clipboard contents or database contents in reports, fixtures, screenshots,
or logs.

## Repository map

| Path | Responsibility |
| --- | --- |
| `main.go` | Initializes shared application state and selects GUI or watcher mode. |
| `app.go` | Holds application metadata, resolves XDG data/config directories, and initializes config and SQLite. |
| `config.go` | Loads and saves the JSON configuration, currently including `close_on_copy`. |
| `database.go` | Opens `clyp.db`, creates the initial schema, tracks query/search state, and vacuums/closes the database. |
| `clipboard.go` | Reads/writes clipboard text and images, persists history, searches, copies, deletes, and updates recency. |
| `watcher.go` | Runs the held GTK watcher application and owns clipboard monitoring. |
| `ipc.go` | Implements GUI refresh notifications over `/tmp/clyp.sock`. |
| `gui.go` | Builds the GTK UI, renders history rows, handles keyboard/mouse behavior, actions, styling, autostart, and watcher launch. |
| `resources/ui/main.ui` | Embedded GTK Builder UI and menu/action declarations. |
| `resources/css/style.css` | Embedded application CSS. |
| `resources/clyp-watcher.desktop` | Embedded autostart entry written into the user's config directory. |
| `data/bio.murat.clyp.desktop` | Installed desktop launcher for the GUI. |
| `data/bio.murat.clyp.metainfo.xml` | Installed AppStream metadata. |
| `data/icons/hicolor/` | Installed application icons at the supported sizes. |
| `.goreleaser.yaml` | Linux `amd64` CGO archive build and release attachment configuration. |
| `nfpm.yaml` | DEB, Arch Linux, and RPM package metadata and installed-file mapping. |
| `bio.murat.clyp.json` | Flatpak manifest, including its pinned source commit and vendored build. |
| `build.sh` | Local GUI/watcher helpers plus snapshot and release orchestration. |
| `vendor/` | Tracked generated dependency tree; do not hand-edit it. |
| `architecture-1.png`, `screenshot-1.png` | README documentation assets. |

Ignored local outputs include `clyp`, `dist/`, `build-dir/`, and
`.flatpak-builder/`. Do not add these artifacts to commits.

## Runtime architecture and lifecycle

### Shared initialization

`app.init()` runs before mode selection. It creates the application data and
config directories, initializes the JSON config, and opens SQLite in both GUI
and watcher processes. Directory locations come from GLib's XDG helpers:

- Data directory: `${XDG_DATA_HOME:-~/.local/share}/bio.murat.clyp/`
- Database: the `clyp.db` file inside that data directory
- Config directory: `${XDG_CONFIG_HOME:-~/.config}/bio.murat.clyp/`
- Config file: the `config.json` file inside that config directory

Do not replace XDG-aware paths with hard-coded home-directory paths. Tests must
not point these globals at the user's real data.

### Mode selection

`main.go` currently joins `os.Args` and recognizes both `--watch` and the
deprecated `watch` spelling with substring checks. `gui.startWatcher()` passes
an argument containing a leading space, which works with this parser. This is
unusual but currently coupled behavior. If argument parsing is changed, update
the launcher and both supported spellings together, then verify that the GUI
actually starts watcher mode rather than another GUI process.

Keep the application IDs stable unless the task explicitly requires a product
identity migration. GTK uses `bio.murat.clyp` for the GUI and
`bio.murat.clyp_watcher` for the watcher; changing either affects single-instance
behavior and process activation.

### GUI process

At activation, the GUI:

1. Loads embedded Builder XML and CSS.
2. Queries and renders the newest clipboard records.
3. Registers window, search, shortcut, menu, theme, startup, and close-on-copy
   behavior.
4. Starts or activates the watcher with `GDK_BACKEND=x11`.
5. Starts the Unix-socket listener in a goroutine.

GTK object mutation belongs on the GTK main context. When adding background
work, marshal UI changes back to the main context instead of directly touching
widgets from arbitrary goroutines. Take particular care when modifying the IPC
listener, because it currently triggers GUI refreshes from its listener path.

The XML IDs used by `gui.go` are contracts: `gtk_window`, `clipboard_list`,
`search_entry`, `search_bar`, `search_toggle_button`, and `shortcuts`. Menu
actions in XML must remain aligned with the `app.*` actions registered in Go.
Renaming or moving any embedded file also requires updating the matching
`//go:embed` directive.

### Watcher process

The watcher is a held GTK application without a window. On activation it
vacuums SQLite, reads the most recently inserted content for adjacent-duplicate
suppression, and subscribes to GDK clipboard changes. It is intentionally forced
to the X11 backend by both the GUI launcher and autostart desktop entry, even
when the GUI itself is native Wayland. Preserve that split unless the task is
specifically about clipboard backend support and it is tested on the relevant
display servers.

When offered formats contain both text and image data, current logic gives text
priority. Text is trimmed before storage. Images are converted to PNG and then
base64-encoded into the SQLite `TEXT` column. Do not change format priority,
normalization, or encoding as an incidental refactor.

### IPC

The GUI owns `/tmp/clyp.sock`: it removes a stale path, listens, reads one byte,
reloads the list, and focuses the first row. The watcher connects, writes `1`,
and closes. There is no versioned payload or request/response protocol.

Any IPC change must update both `notify()` and `listen()`. Test stale-socket
startup, absent-GUI behavior, repeated notifications, shutdown, multiple
instances, and same-machine multi-user implications. Do not delete arbitrary
socket paths or terminate processes that the current task did not start.

## Persistent data contracts

The current SQLite table is `clipboard` with these semantic fields:

- `id`: autoincrementing primary key
- `type`: `1` for text and `2` for a PNG image encoded as base64 text
- `date_time`: SQLite `CURRENT_TIMESTAMP`, also used for newest-first ordering
- `content`: trimmed text or base64 PNG data

Preserve the following visible behavior unless the task explicitly changes it:

- The unfiltered list shows at most 30 rows, newest first.
- The window title shows visible rows versus the total database row count.
- Search is a parameterized `LIKE` query, is limited to 30 rows, and searches
  text items only.
- Copying an item updates its timestamp, which moves it to the top.
- Immediately repeated clipboard content is suppressed in memory.
- Before inserting an image, old image rows are pruned so no more than three
  images remain after the insert.
- Pressing Enter or double-clicking copies the selected item; Delete removes it.
- Escape hides active search first and otherwise closes the GUI.
- `CloseOnCopy` is persisted in JSON and applies to keyboard and double-click
  copy actions.

There is no migration framework. Editing only the initial `CREATE TABLE` block
is not sufficient for users who already have `clyp.db`. Any schema change must
include an idempotent, transactional migration path for existing databases and
must preserve clipboard data. Never solve a schema problem by deleting or
recreating a user's database. Test simultaneous GUI/watcher access and database
locking when changing schema, indexes, pragmas, transactions, or `VACUUM`.

Use bound SQL parameters for values. New database operations must check and
report errors, close rows, and check row iteration errors. Logs may identify an
operation or record ID, but must not contain clipboard content.

## Autostart behavior

The `Run on Startup` action controls
`~/.config/autostart/clyp-watcher.desktop`. The file's content is embedded from
`resources/clyp-watcher.desktop`, and an existing entry is rewritten when the
GUI activates. The entry runs `env GDK_BACKEND=x11 clyp --watch`, so it assumes
the packaged executable is on `PATH`.

Autostart changes affect a real login session. Tests must not overwrite or
remove an entry they did not create. If this area is changed, verify missing
directories, add/remove behavior, upgrades of an existing entry, executable
resolution, and login-session startup. Keep the embedded resource and Go code
in sync.

## Development environment

Use the Go version declared by `go.mod`. Do not take the README's prose version
as authoritative if the two differ. CGO must remain enabled. On Debian-family
systems, the README lists the GTK/GLib/Graphene/Cairo/Pango/GdkPixbuf development
packages and a C build toolchain needed by gotk4. Do not run `sudo`, install
system packages, or mutate the developer's machine without explicit approval.

The standard build path is vendored and should work without modifying module
metadata:

```bash
go test -mod=vendor ./...
go vet -mod=vendor ./...
clyp_build_dir="$(mktemp -d)"
go build -mod=vendor -o "$clyp_build_dir/clyp" .
```

There are currently no repository-owned `*_test.go` files, so `go test` is
primarily a full compile/link check. The first clean gotk4/CGO build can take
several minutes and may emit compiler warnings from generated vendored bindings;
judge success by the command's exit status and report warnings separately.

For a focused Go-only edit, run at minimum:

```bash
gofmt -w <changed-go-files>
go test -mod=vendor ./...
git diff --check
```

Run `go vet -mod=vendor ./...` and a separate `go build` for changes that affect
behavior, concurrency, database access, CGO integration, or release output. Do
not claim a successful build if only formatting or source inspection was run.

### UI and metadata validation

Use the validators relevant to the files changed:

```bash
gtk4-builder-tool validate resources/ui/main.ui
desktop-file-validate data/bio.murat.clyp.desktop resources/clyp-watcher.desktop
appstreamcli validate --no-net data/bio.murat.clyp.metainfo.xml
```

At this revision, GTK Builder and desktop-entry validation are clean. AppStream
validation has an existing warning for a missing homepage URL plus informational
and pedantic metadata findings, so it exits nonzero. Separate that baseline from
new diagnostics; when the task touches AppStream metadata, resolve applicable
findings rather than expanding the baseline.

XML/CSS validation is not visual verification. For UI changes, run the native
application in a graphical Linux session and inspect the default 500x600 window,
long text, image previews, empty/search-result states, keyboard focus, light and
dark styles, and narrow layouts. Do not update `screenshot-1.png` unless the task
requires documentation imagery, and inspect the resulting image before claiming
it is correct.

### Manual runtime testing precautions

Running either mode reads the live clipboard, writes a database, may create a
socket, and may interact with an existing watcher. Prefer compile-time checks
unless runtime behavior is in scope. Before a live smoke test:

- Explain that clipboard history will be observed and stored.
- Use dedicated XDG data/config directories where practical.
- Use synthetic, non-sensitive clipboard samples.
- Record and stop only the processes started by the test.
- Restore only autostart or clipboard state that the test itself changed.
- Check for an orphaned test watcher and stale test socket afterward without
  disrupting an already-running user instance.

`RUN_ENV=dev` makes the GUI launch `./clyp`; without it, the GUI launches `clyp`
from `PATH`. The helper script expects `dist/clyp_linux_amd64_v1/` to exist:

```bash
mkdir -p dist/clyp_linux_amd64_v1
./build.sh run-gui
./build.sh run-gui theme-light
./build.sh run-watcher
```

These commands are interactive and affect clipboard state; do not use them as
unattended validation. When watcher/backend behavior changes, test the supported
X11 and Wayland/XWayland scenarios explicitly and state which were actually run.

## Testing guidance for new code

- Add focused `*_test.go` coverage for non-GTK logic when feasible.
- Use `t.TempDir()` for config and database fixtures. Save and restore global
  state, and close temporary database handles in cleanup functions.
- Never open the user's real `clyp.db` from a test.
- Avoid the shared `/tmp/clyp.sock` in parallel tests; inject or derive a unique
  test socket path before adding IPC tests.
- Do not assume GTK tests are headless. Separate pure transformations and query
  logic from display-dependent code when testability is part of the task.
- For database changes, cover a fresh database, an existing database migration,
  retained records, search/order semantics, and concurrent process access.
- For clipboard changes, cover empty content, adjacent duplicates, Unicode text,
  large text, invalid image data, and the three-image retention rule as relevant.

## Code and change conventions

- Run `gofmt` on every changed Go file. Follow normal Go naming and keep imports
  grouped by the formatter.
- The current design uses small receiver methods over package-level singleton
  state. Do not introduce a broad architecture rewrite for a scoped fix.
- Prefer explicit errors and contextual `log.Printf` messages over silent
  failure. Reserve panics for unrecoverable startup invariants.
- Keep clipboard content out of logs and error messages.
- Keep SQL parameterized and make multi-step mutations transactional when
  partial completion would corrupt state.
- Preserve GTK main-thread requirements and asynchronous callback lifetimes.
- Keep UI behavior keyboard-first. Check focus after list refresh, search,
  deletion, copying, and empty results.
- Preserve existing user-facing text, shortcuts, IDs, ordering, dimensions, and
  adjacent behavior unless the request changes them.
- Mark new GTK Builder user-facing strings translatable where appropriate.
- Keep embedded resources as the runtime source; editing an unrelated installed
  copy will not change the binary.
- Avoid unrelated dependency upgrades, formatting churn, renames, or generated
  asset changes.
- Do not manually edit files under `vendor/`. For an authorized dependency
  change, update module metadata, run `go mod tidy`, run `go mod vendor`, and
  review `go.mod`, `go.sum`, `vendor/modules.txt`, and the vendor diff together.
- Preserve unrelated working-tree changes. Never discard or overwrite user work.

## Packaging, versions, and releases

Release/version information is deliberately spread across several mechanisms.
Before a version change, search the repository rather than updating only the
first match. At minimum inspect:

- `app.go` for the About-dialog version
- `README.md` for download URLs, package filenames, installation commands, and
  the documented Go version
- `nfpm.yaml` for package version and metadata
- Git tags, from which GoReleaser derives its release version
- `bio.murat.clyp.json` for the pinned Flatpak source commit, runtime, SDK, and
  architecture

Keep application metadata, package artifacts, docs, and tag naming consistent.
Do not update the Flatpak source commit to an unpushed or nonexistent revision.
When changing installed paths or the application ID, update desktop files,
AppStream metadata, icon destinations, embedded resources, Go code, and package
manifests as one coordinated change.

`./build.sh package-snapshot` requires GoReleaser and nFPM and cleans/recreates
parts of `dist/`. Run it only when package validation is in scope. Inspect the
resulting DEB, Arch, and RPM package contents and metadata; successful command
exit alone does not prove that launchers, icons, or dependencies are correct.

`./build.sh release` is a publishing operation, not a validation command. It
performs broad text replacement, stages the whole worktree, commits, tags,
pushes, builds packages, and publishes a GitHub release. Never run it, create a
tag, push, publish, or use a release token unless the user explicitly requests
that exact release action. Before an authorized release, require a clean and
reviewed tree, verify old/new version arguments, inspect the staged diff, and
keep credentials out of command output and repository files.

## Contribution and documentation expectations

The README asks contributors to open an issue before submitting code changes.
Do not create issues, commits, pull requests, tags, or releases unless requested.
When behavior changes, update README usage, shortcuts, architecture, build, or
installation text in the same change when relevant. Keep claims grounded in
code and actual validation; distinguish local compilation, native GUI testing,
package inspection, and release/published verification.

## Definition of done

Before handing off a change:

1. Re-read the request and inspect the final diff for scope and preserved
   behavior.
2. Run `gofmt` on changed Go files and `git diff --check`.
3. Run `go test -mod=vendor ./...`; run vet and a separate build when warranted.
4. Run GTK Builder, desktop-entry, and AppStream validators for touched files.
5. Add or update focused tests for changed non-GUI logic.
6. Perform native visual/runtime checks for GUI, watcher, clipboard, IPC,
   autostart, or compositor-sensitive changes when safe and in scope.
7. Verify data migration and preservation for every schema change.
8. Verify all version and packaging surfaces for release-related changes.
9. Confirm that ignored binaries/packages, live clipboard data, local databases,
   credentials, and unrelated files are absent from the diff.
10. Report exactly what passed, what produced baseline warnings, and what could
    not be run; never imply live GUI, package, or release verification from a
    source-only check.
