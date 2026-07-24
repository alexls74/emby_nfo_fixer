# Emby NFO Fixer
🌐 English | 🇷🇺 [Русский](README.md)

**Emby NFO Fixer** is a CLI utility for validating, fixing, and automatically enhancing metadata in Kodi NFO files for seamless use with the Emby media server.

## Features 

* **Automatic Tag Fixes:** Validates and fixes the structure and formatting of metadata in NFO files.
* **Premiered Date Fetching (TMDB API):** Automatically reads the `<tmdbid>` tag and fetches missing premiered dates from The Movie Database (TMDB) if the `<premiered>` tag is absent.
* **Emby Library Rescan:** Optionally triggers a library scan in Emby if any NFO files were modified.
* **Data Safety:** Automatically creates backups of original files prior to modification while preserving the folder directory structure.
* **Detailed Logging:** Generates separate log files (`changed.log`, `skipped.log`, `error.log`) inside the backup directory.
* **Multilingual Support:** Native support for English and Russian languages.
* **Silent Mode:** Enables execution without console output.

---

## Installation

### Pre-built Binaries (Recommended)

Download the appropriate archive for your operating system from the **[Releases](https://github.com/alexls74/emby_nfo_fixer/releases)** section:
* **Windows:** `emby_nfo_fixer_vXX.XX.XX_windows_x64.zip` (x64)
* **macOS:** `emby_nfo_fixer_vXX.XX.XX_macOS_universal.zip` (Universal Binary)
* **Linux:** `emby_nfo_fixer_vXX.XX.XX_linux_amd64.tar.gz` (amd64)

Extract the archive to any directory of your choice.

### Building from Source

* Download and install **[Go](https://go.dev/doc/install)**.
* Clone the repository:
```bash
git clone [https://github.com/alexls74/emby_nfo_fixer.git](https://github.com/alexls74/emby_nfo_fixer.git) && cd emby_nfo_fixer
```
* Compile the application
```bash
make build
```
The compiled binary executable will be available in the **_/build_** directory.

## Configuration File

On first launch, a `config.conf` file is automatically created in the user's home directory.
The exact path to the configuration file can be viewed in the help output or by running the executable with the **[-v]** flag.

## TMDB API Setup

The utility relies on **The Movie Database (TMDB)** to automatically populate missing premiere dates.

1. On initial startup, the wizard will prompt you to enter a TMDB API token.
2. The provided token will be validated and saved to `config.conf`.
3. If skipped or left blank, TMDB integration will be disabled. You can manually set this up in the configuration file later.

> **How to get a TMDB API Token:**
> 1. Sign up on **[The Movie Database (TMDB)](https://www.themoviedb.org/)**.
> 2. Go to **Profile and Settings → API Subscription**.
> 3. Generate an API Key.
> 4. Copy the **API Read Access Token**.

## Emby API Setup

To automatically trigger an Emby library scan after processing, the tool uses the **[Emby REST API](https://dev.emby.media/doc/restapi/index.html)**.

1. Running with the **[-e]** option requires specifying both the source and backup directories.
2. On initial startup, you will be prompted to enter your Emby server details.
3. The server URL can be local (**`http://192.168.1.199:8096`**) or external (**`https://domain.tld`**).
4. The URL and API Key will be saved to `config.conf`.

> **How to get an Emby API Key:**
> 1. Open the Emby Server Admin Console.
> 2. Navigate to **API-ключи**.
> 3. Generate a new key.

## Usage

Run the utility from the command line by passing the required paths for the source NFO directory and the backup destination.

Default usage (no extra flags):
```bash
emby_nfo_fixer /mnt/Movies /mnt/Backup
```
Process NFO files and trigger an Emby library rescan if changes were made:
```bash
emby_nfo_fixer -e /mnt/Movies /mnt/Backup
```
Silent mode (suppresses console output):

```bash
emby_nfo_fixer -s /mnt/Movies /mnt/Backup
```

---

## Roadmap

- [x] Structure validation and tag placement fixes.
- [x] Checking and adding the `<premiered>` tag.
- [X] Optional Emby library rescan triggering.
- [x] Multilingual interface support.