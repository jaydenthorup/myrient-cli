# 🎮 myrient-cli

`myrient-cli` is a terminal-based interactive CLI tool for browsing and downloading retro game dumps from the [Myrient Redump mirror](https://myrient.erista.me/files/Redump/). It allows you to navigate platforms, search for titles, queue multiple downloads, and automatically extract the games.

---

## ✨ Features

- Interactive menu to select a gaming platform (PS2, GameCube, Dreamcast, etc.)
- Live search/filter by game title, with regex support
- Paginated browsing
- Queue and download games sequentially for reliability
- Summary messages for queued downloads
- Pretty progress bars (download + unzip)
- Automatic `.zip` extraction and cleanup
- Customizable download directory

---

## 🧰 Requirements

- Go 1.22+
- Terminal (Linux, macOS, Windows)

---

## ⚙️ Build

In the project root, run:

```bash
make
```

This will build binaries for:

- Linux (amd64, arm, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

Binaries are placed in the `builds/` directory.

To clean old builds:

```bash
make clean
```

---

## ▶️ Run

From terminal:

```bash
./myrient-cli
```

The CLI will walk you through:

1. Selecting a platform
2. Filtering titles (supports regex: e.g. `(?i)halo.*usa`)
3. Browsing the game list (with pagination)
4. Queuing downloads (all, or by number)
5. Watching progress as files are downloaded and extracted
6. Viewing summary messages for queued downloads
---

## 🆕 Improvements

- Regex filtering for advanced search
- Sequential download queue (no more concurrency errors)
- Clear summary messages after queuing downloads
- All map accesses are now concurrency-safe

---

## 📁 Download Directory

By default, games are downloaded to:

```
./.downloads/
```

To override this path, set an environment variable:

```bash
export MYRIENT_DOWNLOADS_PATH=/path/to/your/folder
```

The tool will automatically create the folder if it doesn't exist.

---

## 📄 License

MIT – feel free to use, modify, and share.
