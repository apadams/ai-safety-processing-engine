[![TWSLogo](https://github.com/Tobiwan-Cloud-Solutions/images/blob/main/TWSBannerGHLogo.png)](https://apadams.github.io)

# AI Safety Blocklist (Community Edition)

> **Status:** Active & Maintained
> **License:** CC BY-NC-SA 4.0
> **Update Frequency:** Weekly

## Overview
The **AI Safety Blocklist** is an open-source intelligence (OSINT) feed tracking high-risk, unverified, and "Shadow IT" AI applications.

This list is designed for security engineers, network administrators, and privacy-conscious individuals who wish to block data exfiltration to unvetted AI tools. It targets:
* **Shadow SaaS:** AI tools launched on Reddit/ProductHunt with no compliance documentation.
* **Data Scrapers:** Bots and agents that aggressively harvest data.
* **High-Risk Jurisdictions:** AI tools hosted on non-compliant infrastructure.

## What's Included (Lite vs. Pro)

| Feature | **Community Edition** (This Repo) | **Enterprise Feed** (Coming Soon) |
| :--- | :--- | :--- |
| **Volume** | Top 500 (High Confidence) | Full Database (5,000+) |
| **Update Speed** | Weekly | Real-time (Hourly) |
| **Metadata** | Domain Only | IP, Hosting Provider, Risk Score |
| **Format** | `.txt` (Pi-hole compatible) | JSON / CSV / API |
| **Support** | Community Issues | SLA Support |

## Usage

### Pi-hole / AdGuard Home
1.  Go to your Adlists / Blocklists settings.
2.  Add the following URL:
    ```
    https://raw.githubusercontent.com/Tobiwan-Cloud-Solutions/ai-safety-blocklist/main/shadow-ai-lite.txt
    ```

### Palo Alto / Fortinet (External Dynamic List)
*Note: Enterprise firewalls require the clean, IP-resolved feed found in our Enterprise tier to avoid false positives.*

## Disclaimer
This list is generated via automated analysis of public launch data (Hacker News, Reddit, GitHub). While we strive for accuracy, false positives may occur. Use at your own risk.

**[Submit a False Positive](https://github.com/Tobiwan-Cloud-Solutions/ai-safety-blocklist/issues)**

### If you like this project, please consider supporting me.
[![Buy Me A Coffee](https://img.shields.io/badge/Fuel_the_Engine-FFDD00?style=for-the-badge&logo=buymeacoffee&logoColor=black)](https://www.buymeacoffee.com/apadams)

