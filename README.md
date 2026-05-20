# OffCourse

## Overview

OffCourse is a local course management application that enables you to view, organize, and track progress through educational content
stored on your local filesystem. It provides a web-based interface for browsing courses, tracking learning progress, and managing course
materials without requiring an internet connection.

The application automatically scans course directories to identify assets (videos, PDFs, markdown files and text files) and attachments,
organizing them into structured lessons with progress tracking capabilities.

## Architecture

### Frontend

- **SvelteKit** with TypeScript for the web interface
- **BitsUI** for the web interface components
- **Tailwind CSS** for the CSS
- **Vidstack** for the video playback

### Backend

- **Go** application with RESTful API
- **SQLite** (`data.db`, `logs.db`) via [`github.com/mattn/go-sqlite3`](https://github.com/mattn/go-sqlite3) and
  [`github.com/jmoiron/sqlx`](https://github.com/jmoiron/sqlx)
- **CGO** for SQLite and course card WebP encoding (see [Prerequisites](#prerequisites))

#### Data Directory

A data directory is created automatically when the application is first launched.

By default, the `oc_data` directory will be created relative to where the binary is launched, however,
this can be overridden with `--data-dir xxx`

The purpose of this directory is to store application related information, such as the sqlite databases and
transcoded video files

**Database**

The following 2 `*.db` will be created in the data directory

- `data.db` - Main application data (courses, users, progress)
- `logs.db` - Application logs

**Video Transcoding**

`offcourse` provides on-demand video transcoding for HLS streaming

The transcoded videos will be placed in the data directory under `hls`

**Card Optimization**

`offcourse` automatically optimizes course card images during course scanning

Card images (`card.jpg`, `card.jpeg`, `card.png`, `card.webp`, `card.tiff`) are converted to WebP using libwebp, scaled to a
maximum width of 800px (aspect ratio preserved), and encoded at 85% quality

The optimized card is only kept when the WebP file is smaller than the original; otherwise the original file is served

The optimized card images will be placed in the data directory under `cards`

- Each course's optimized card is stored as `{course-id}.webp`
- A fallback card (`fallback.webp`) is used when a course has no card image
- Optimized cards are automatically deleted when a course is deleted

## Build and Run

### Manual

#### Prerequisites

- Node.js >= 22.12.0
- pnpm >= 8
- Go >= 1.22.4
- Make
- A C toolchain (required for CGO; Xcode Command Line Tools on macOS, `build-essential` on Debian/Ubuntu)
- **SQLite** development libraries (for `mattn/go-sqlite3`; e.g. `libsqlite3-dev` on Debian, `sqlite` via Homebrew on macOS)
- **libwebp** development headers (for course card encoding; e.g. `libwebp-dev` on Debian, `webp` via Homebrew on macOS)
- **FFmpeg** and **FFprobe** (for video processing and HLS transcoding only; not used for course cards)

The backend must be built with CGO enabled (`CGO_ENABLED=1`, which is the default on most systems). Without CGO or the libraries
above, `go build` and `go test` will fail when linking SQLite or WebP support

#### Build

The `make build` command will build both the frontend and backend, automatically including version and commit information in the binary

The output binary will be `offcourse` in the root directory

Note: The frontend must be built first as the contents of `ui/build` are embedded into the Go binary

**Build Command**

```bash
make build
```

**Building for Different Platforms**

To build for a particular distro/arch, use `GOOS` and `GOARCH` environment variables:

```bash
GOOS=linux GOARCH=amd64 make build
```

#### Run

**Basic Run**

```bash
./offcourse serve
```

**Overrides**

- Sets the port to `8080` (default `9081`)
- Sets the data directory to `my-data` (default `oc_data`)
- Enabled signing up (default `disabled`)

```
./offcourse serve --http 0.0.0.0:8080 --data-dir ./my-data --enable-signup
```

### Docker

The Docker image builds the backend with `CGO_ENABLED=1` and installs SQLite and libwebp development packages. The runtime
image includes FFmpeg, libwebp, and SQLite libraries

#### Prerequisites

- Docker

#### Build

```bash
docker build --platform linux/amd64 -t offcourse:test -f docker/Dockerfile .
```

#### Run

See the [Docker README](docker/README.md), which explains how to generate a docker compose file, mount volumes,
and more

## Development

The following will run the ui in `dev` mode in 1 terminal and the backend golang application in another terminal. Any changes to either
backend or frontend code will result in automatic reloading/rebuilding

### Prerequisites

- [air](https://github.com/air-verse/air)
- The same [native build prerequisites](#prerequisites) as a manual build (CGO, SQLite, libwebp). FFmpeg is only required for video features

### Run

**Frontend**

1.  Open a terminal

2.  Move into the ui directory

    ```bash
    cd ui
    ```

3.  Install the package dependencies

    ```bash
    pnpm install
    ```

4.  Run the dev server

    Note: Ignore the address given here. Everything runs through the go application

    ```bash
    pnpm run dev
    ```

**backend**

1. Open another terminal

2. Install the go dependencies

   ```bash
   go mod download
   ```

3. Run air

   Note: Defaults can be overridden, for example `air -- --http 0.0.0.0:8080`

   ```bash
   air
   ```

4. In a browser, open the address given in the line `Bootstrap required: ...`

   Note: If already bootstrapped, in a browser, open the address given in the line `Server started at ...`

### Go Tests

The Go application includes a suite of tests that can be run using the `go test` command

From the root of the project:

```bash
go test -tags dev -v ./...
```

- **`-tags dev`** — uses a stub UI embed so tests do not require building `ui/build` first (see [Build](#build))
- **CGO** — same requirements as a normal build (SQLite + libwebp). CI installs `libsqlite3-dev` and `libwebp-dev` alongside FFmpeg

If you already have `ui/build` from a frontend build, `go test ./...` without `-tags dev` also works.

## CLI Commands

OffCourse includes several CLI commands

### Serve

The `serve` command runs the application

```bash
./offcourse serve [options]
```

#### Options

- `--http <address>` - HTTP server address (default: 127.0.0.1:9081)
- `--data-dir <path>` - Data directory path (default: ./oc_data)
- `--enable-signup` - Allow user registration
- `--dev` - Run in development mode
- `--debug` - Enable debug logging

### Admin

The `admin` command allows you to reset the password of a user

```bash
./offcourse admin reset-password <username>
```

## Bootstrapping

When first launched, OffCourse needs to be bootstrapped with an initial administrator account

### Process

1. When the application starts, it checks if any admin users exist in the database

2. When no admin users are found, a secure bootstrap URL is displayed in the console

   ```shell
   ⚠️  Bootstrap required: http://127.0.0.1:9081/auth/bootstrap/[unique-token]
   Token expires in 5 minutes
   ```

3. Visit the bootstrap URL to create your administrator account

## Adding Courses

### Overview

Courses are organized as directories containing files

When a course is added, a scan of the directory is automatically run to identify assets and attachments, building out a module/chapter
and lesson structure

Typically, 1 asset == 1 lesson, however, assets may be grouped such that n assets == 1 lesson

Assets are files whereby the filename contains a prefix, title and extension. For example, `01 Introduction.mp4`

Attachments are files whereby the filename contains a prefix, with an optional title and extension. For example, `01 Extra.url` and is
linked to an asset via a shared prefix

### Example Course

The following is an example of a course directory structure

```
My Course/
├── card.jpg                   # Course card image
├── Chapter 1/                 # Module (chapter)
│   ├── 01 Overview.mp4        # First asset
│   ├── 01 Overview Notes.txt  # Attachment for first asset
│   └── 02 Example.md          # Second asset
└── Chapter 2/
    ├── 01 Deep Dive.pdf       # First asset
    └── 01 Source Links.txt    # Attachment for first asset
```

### Course Card

A course card is an image named `card.xxx` at the root of a course directory

The extension may be one of `.jpg`, `.png`, `.webp`, `.tiff`

### Assets and Attachments

#### Assets

Assets are primary course materials, such as videos, PDFs, markdown and text

A file is identified as an asset when it matches the following filename pattern and contains a supported asset extension. See
[Supported Asset Types](#supported-asset-types) for the list of supported extensions

Pattern:

(`*` means optional)

```
[prefix] [separator *] [title].[extension]
```

_required_

- `prefix`: A number such as `1`, `01`, `001`
- `title`: Any valid characters
- `extension`: A supported asset extension

_optional_

- `separator`: One of `.` or `-`. For example, `01.` or `1 -`

Examples:

- `01 Introduction.mp4`
- `02. Advanced Concepts.pdf`
- `3 - Getting Started.md`

#### Attachments

Attachments are supplementary materials linked to an asset via a shared prefix

A file is identified as an attachment when it matches the following filename pattern and is **not** an asset

Pattern:

(`*` means optional)

```
[prefix] [separator *] [title *].[extension *]
```

_required_

- `prefix`: A number such as `1`, `01`, `001`

_optional_

- `separator`: One of `.` or `-`. For example, `01.` or `1 -`
- `title`: Any valid characters
- `extension`: A supported asset extension

Examples:

- `01`
- `01 Notes.txt`
- `01 Introduction Notes.pdf`

#### Grouped Assets

Assets may be grouped together using a `sub-prefix` and an optional `sub-title`

Grouped assets become 1 lesson, meaning the assets will be rendered in the order of the sub-prefix on the same lesson page

Pattern:

Note: \* == optional

```
[prefix] [separator *] [title] {[sub-prefix] [sub-separator *] [sub-title *]}.[extension]
```

_required_

- `prefix`: A number such as `1`, `01`, `001` (required)
- `title`: Any valid characters
- `sub-prefix`: A number such as `1`, `01`, `001`
- `extension`: A supported asset extension

_optional_

- `separator`: One of `.` or `-`. For example, `01.` or `1 -`
- `sub-separator`: One of `.` or `-`. For example, `01.` or `1 -`
- `sub-title`: Any valid characters

Examples:

- `01 Introduction {1 Part 1}.mp4`
- `01 Introduction {2 - Description}.md`
- `01 Introduction {03 Part 3}.mp4`
- `01 Introduction {04}.mp4`

### Supported Asset Extensions

The following are the supported asset extensions, categorized by type

**Video**

- `.mp4`, `.avi`, `.mkv`, `.webm`, `.ogv`

**Audio**

- `.mp3`, `.m4a`, `.ogg`, `.wav`, `.flac`

**Documents**

- `.pdf`
- `.md`
- `.txt`

### Asset Priority

When multiple file in a directory share the same prefix and a supported asset extension but without a sub-prefix, we use a priority
list to determine which file will be marked as the asset and which will be marked as the attachment(s)

1. **Video** (highest priority)
2. **PDF**
3. **Markdown**
4. **Text** (lowest priority)

For example, If you have both `01 Introduction.mp4` and `01 Introduction.md`, the video file will be marked as the asset and the markdown
file will be marked as the attachment
