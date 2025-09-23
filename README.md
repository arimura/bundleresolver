# Bundle Resolver

Bundle Resolver is a small CLI that reads iOS App Store numeric IDs and/or Google Play package names from STDIN (one per line) and prints tab-separated (TSV) lines containing the app name, publisher name, and store URL. You can choose which fields to output; by default all three are printed in the order: `name`, `publisher`, `url`.

## Features

- Bulk resolve many IDs via pipe / redirected file
- Automatic platform detection
	- All digits -> treated as an iOS App Store app ID
	- Contains a dot -> treated as a Google Play package name
- Clean TSV output (easy to post-process in shell / scripts)
- Reorder or subset output columns with `--fields`
- Continues on partial failures (errors go to STDERR, successes still emitted)

## Install

Requires Go 1.22+.

```bash
go install github.com/arimura/bundleresolver/cmd/bundleresolver@latest
```

Ensure `$GOPATH/bin` (or your `GOBIN`) is on `PATH`.

## Usage

### Basic

```bash
echo "123456789" | bundleresolver
```

```bash
cat ids.txt | bundleresolver
```

Example `ids.txt`:

```
123456789
com.example.myapp
987654321
```

### Select specific fields

```bash
echo "123456789" | bundleresolver --fields name,url
```

```bash
echo -e "123456789\ncom.example.myapp" | bundleresolver -f publisher
```

### Post-process with standard UNIX tools

```bash
echo "123456789" | bundleresolver | cut -f1   # only name column
```

## Command Reference

```
bundleresolver [OPTIONS]
```

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--fields <list>` | `-f` | Comma-separated list of fields to output (order preserved). Allowed: `name,publisher,url` | `name,publisher,url` |
| `--version` | (none) | Print version and exit | (off) |
| `--help` | `-h` | Show help | (off) |

### Field definitions

| Field | Meaning |
|-------|---------|
| `name` | App display name |
| `publisher` | Developer / publisher name |
| `url` | Official store page URL |

## Output Format (TSV)

One record per input line. Columns are separated by a single TAB (`\t`). No trailing TAB. Unavailable values become empty strings. The column count always matches the number of requested fields.

Example (default 3 fields):

```
AppName	PublisherName	https://apps.apple.com/app/id123456789
My Android App	Sample Studio	https://play.google.com/store/apps/details?id=com.example.myapp
```

