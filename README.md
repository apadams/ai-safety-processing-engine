[![TWSLogo](https://github.com/Tobiwan-Cloud-Solutions/images/blob/main/TWSBannerGHLogo.png)](https://apadams.github.io)

# ⚙️ AI Safety Pro Engine

> **⚠️ CONFIDENTIAL:** This repository contains the source code, raw data, and automation logic for the SentinelFlow threat intelligence product. **Do not make this repository public.**

## 🔍 System Overview
This "Engine" is an automated pipeline that ingests, filters, and enriches threat intelligence data regarding "Shadow AI" applications. It serves two products:
1.  **The Asset (Internal):** `master_threat_db.csv` - The full, enriched database with IP resolution and risk scores.
2.  **The Community Feed (Public):** A sanitized "Lite" list pushed weekly to the [Public Repo](https://github.com/Tobiwan-Cloud-Solutions/ai-safety-blocklist).

## 🏗️ Architecture

| Module | Path | Function |
| :--- | :--- | :--- |
| **Ingestor** | `ingest/main.go` | Scrapes Hacker News (API), Reddit (RSS), and GitHub (API). Handles backfilling and daily deltas. |
| **Sanitizer** | `utils/sanitizer.go` | **CRITICAL.** Strips proxy URLs, removes protocols, and enforces the `allowlist`. |
| **Filter** | `filter/main.go` | Assigns Risk Scores (0-100) and resolves IP addresses/Hosting Providers. |
| **Publisher** | `publisher/main.go` | Generates the `shadow-ai-lite.txt` file for the public repo (High Risk only, No IPs). |
| **Config** | `config/allowlist.txt` | The "Do Not Block" list. **Update this to fix false positives.** |

## 🤖 Automation (SentinelBot)

This repo uses **GitHub Actions** to run autonomously.

### 1. The Daily Grind (`.github/workflows/daily-grind.yml`)
* **Schedule:** Every day at 00:00 UTC.
* **Action:** Runs `ingest/main.go --mode=daily`.
* **Output:** Commits new threats to `master_threat_db.csv` inside this repo.

## 🛠️ Local Development 

**Prerequisites:**
* Go 1.21+
* `go mod tidy` (to install dependencies)

**Common Commands:**
```bash
# Run the daily ingestion manually (Dry Run - won't save to DB unless specified)
go run ingest/main.go --mode=daily

# Force a re-publish of the Community List locally
go run publisher/main.go
