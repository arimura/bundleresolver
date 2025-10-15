# Bundle Resolver

Bundle Resolver is a small CLI that reads iOS App Store numeric IDs and/or Google Play package names from STDIN (one per line) and prints tab-separated (TSV) lines containing the bundle column (iOS App ID or Android package name), app name, publisher name, and store URL. You can choose which fields to output; by default all four are printed in the order: `bundle`, `name`, `publisher`, `url`.

By default a header row (field names) is printed as the first line. Disable it with `--header=false`.

Add `--csv` to switch the delimiter to commas with RFC4180 quoting.

## Features

- Bulk resolve many IDs via pipe / redirected file
- Automatic platform detection
	- All digits -> treated as an iOS App Store app ID
	- Contains a dot -> treated as a Google Play package name
- Clean TSV output (easy to post-process in shell / scripts)
- Optional CSV output with proper quoting
- Reorder or subset output columns with `--fields`
- Continues on partial failures (errors go to STDERR, successes still emitted)
- Option to skip error lines entirely with `--skip-errors`

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

### Emit CSV instead of TSV

```bash
echo "123456789" | bundleresolver --csv
```

When you enable CSV mode, headers and data rows are emitted with commas and double-quoted as needed. Field selection still applies.

### Post-process with standard UNIX tools

```bash
echo "123456789" | bundleresolver | cut -f1   # only name column
```

### Skip error lines

By default, if an ID fails to resolve, an empty (or partial) row is still output to maintain line alignment with the input. To skip failed lines entirely:

```bash
cat ids.txt | bundleresolver --skip-errors
```

Error messages are always written to STDERR regardless of this option.

## Command Reference

```
bundleresolver [OPTIONS]
```

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--fields <list>` | `-f` | Comma-separated list of fields to output (order preserved). Allowed: `bundle,name,publisher,url` | `bundle,name,publisher,url` |
| `--csv` | (none) | Emit output as CSV (RFC4180 with quoting) instead of TSV | `false` |
| `--version` | (none) | Print version and exit | (off) |
| `--header` | (none) | Print header row (`bundle\tname\tpublisher\turl`). Use `--header=false` to suppress | `true` |
| `--skip-errors` | (none) | Skip lines that fail to resolve instead of outputting empty rows | `false` |
| `--help` | `-h` | Show help | (off) |

### Field definitions

| Field | Meaning |
|-------|---------|
| `bundle` | iOS App ID (numeric) or Android package name |
| `name` | App display name |
| `publisher` | Developer / publisher name |
| `url` | Official store page URL |

## Output Format

### TSV (default)

One record per input line. Columns are separated by a single TAB (`\t`). No trailing TAB. Unavailable values become empty strings. The column count always matches the number of requested fields.

Example (default 4 fields):

```
123456789	AppName	PublisherName	https://apps.apple.com/app/id123456789
com.example.myapp	My Android App	Sample Studio	https://play.google.com/store/apps/details?id=com.example.myapp
```

### CSV mode

When `--csv` is supplied, the same data is emitted as comma-separated values with double quotes applied when needed. Newlines are preserved in values but normalized to spaces as usual.

Example:

```
bundle,name,publisher,url
123456789,AppName,PublisherName,https://apps.apple.com/app/id123456789
com.example.myapp,"My Android App",Sample Studio,https://play.google.com/store/apps/details?id=com.example.myapp
```

