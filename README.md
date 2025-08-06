# HKTM

High-performance Discord music bot written in Go, designed to stream audio directly from YouTube using a tightly controlled `yt-dlp → FFmpeg → gopus` pipeline.
---

![HKTM Screenshot](https://cdn.discordapp.com/attachments/1119291447926075412/1402487517567127683/image.png?ex=689417c9&is=6892c649&hm=5b6a2888791c0ffd2ba614f509c310cefdd2186ef6e95abc3393d3392e166d7c&)

> ⚠️ **Work in Progress**  
> This project is developed for personal use in a private Discord server. While the code is open-source for educational purposes, it is **not** production-ready and may lack general support or stability guarantees.

---

## Features

- **Direct Audio Pipeline** — `yt-dlp → FFmpeg → gopus` for full transparency
- **No DCA Dependency** — Avoids DCA’s silent failures and black-box behavior
- **Stream-First Design** — Built for low-latency, stable audio playback
- **Battle-Tested Tools** — Leverages mature tools (FFmpeg, yt-dlp) instead of wrappers

---

## Architecture

The audio streaming flow is purpose-built for performance and clarity:

```mermaid
  A[yt-dlp] --> B[FFmpeg (PCM)]
  B ----------> C[gopus (Opus Encoder)]
  C ----------> D[Discord Voice Channel]
```

### Pipeline Breakdown:

1. **yt-dlp** – Extracts direct audio stream URLs from YouTube.
2. **FFmpeg** – Converts audio to raw PCM format (`48kHz`, `16-bit`, `stereo`).
3. **gopus** – Encodes PCM to Opus (optimized for Discord).
4. **Discord** – Streams Opus frames to voice channels via Discord Gateway.

---

## Why This Approach?

- **Full Pipeline Control** – Know exactly what’s happening at each stage
- **Zero Black Boxes** – Easier to debug and extend than DCA-based solutions
- **Granular Error Recovery** – Handle broken pipes, subprocess failures, etc.
- **Pure Go Integration** – `gopus` enables native Opus encoding

---

## 🛠 Requirements

Make sure the following are installed and available in your `PATH`:

- [Go 1.23+](https://go.dev/dl/)
- [FFmpeg](https://ffmpeg.org/)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [Discord Bot Token](https://discord.com/developers/applications)

---

## 🚀 Getting Started

```bash
git clone https://github.com/latoulicious/Tarumae.git
cd Tarumae
go mod tidy
```

1. Configure your Discord bot token (via `.env` or code).
2. Run the bot:

```bash
go run cmd/main.go
```

---

## Commands

Check the full list of available commands in the [`SLASH_COMMANDS.md`](https://github.com/latoulicious/Tarumae/blob/main/SLASH_COMMANDS.md).

---

## Known Issues

- Audio may **cut off abruptly after prolonged playback** — suspected stream or subprocess timeout. Currently under investigation.

## Made Possible By

Special thanks to [bwmarrin/discordgo](https://github.com/bwmarrin/discordgo) — the foundation that made this bot possible.

- [yt-dlp](https://github.com/yt-dlp/yt-dlp) — For extracting high-quality audio from YouTube
- [FFmpeg](https://ffmpeg.org/) — For reliable audio stream decoding and conversion
- [layeh/gopus](https://github.com/layeh/gopus) — For direct Opus encoding in Go
