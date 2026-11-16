# Pokemon GO Coordinate Monitor (Go Edition)

Pemantau koordinat Pokemon dari `coordinates-api.pokemongopro.com` dan pengirim notifikasi Discord Embed yang sangat ringan, dioptimalkan khusus untuk perangkat dengan sumber daya terbatas seperti **STB (Set-Top Box)** via Docker.

## Fitur Utama

- 🚀 **Sangat Ringan**: Ditulis murni dengan **Go (Zero external dependencies)**. Ukuran image Docker hanya **~15 MB** dan RAM runtime **< 10 MB** (jauh lebih hemat dibanding Python ~180 MB & 35 MB RAM).
- ⚙️ **Konfigurasi Rapi**:
  - Kredensial & Interval diatur lewat file `.env`.
  - Filter payload (daftar Pokemon ID pilihan, species forms, dll.) tersimpan di `config.json`.
- 🔁 **Interval Acak Hemat Resource**: Interval monitoring acak (default `30` hingga `120` menit) agar tidak membebani STB atau memicu rate limit API.
- 💾 **State Persistence**: Menyimpan encounter ID di volume `./data/.state.json` agar Pokemon yang sama tidak dinotifikasi berulang kali.
- 🛑 **Graceful Shutdown**: Mendukung sinyal `SIGTERM` / `SIGINT` (Ctrl+C / `docker stop`) agar kontainer langsung berhenti seketika tanpa harus menunggu timer sleep selesai.

---

## 1. Setup

### a. File `.env` (Kredensial & Interval)
Salin dari template:
```bash
cp .env.example .env
```
Isi file `.env`:
```env
WEBHOOK_URL=https://discord.com/api/webhooks/...
BEARER_TOKEN=eyJhbGciOi...

INTERVAL_MIN=30
INTERVAL_MAX=120
```

### b. File `config.json` (Filter Payload)
File `config.json` sudah diatur dengan daftar `pokemon_ids` dan `species_forms` favorit Anda. Jika ingin mengubah daftar Pokemon di kemudian hari, Anda cukup mengedit `config.json`.

---

## 2. Menjalankan di STB (Docker Compose)

Pastikan direktori `./data` memiliki izin akses:

```bash
mkdir -p data && chmod -R 777 data
```

### Opsi A: Langsung tarik image dari GHCR (Tanpa perlu build di STB - Rekomendasi)
```bash
# Tarik image pre-built multi-arch dari GitHub Container Registry
docker compose pull

# Jalankan
docker compose up -d

# Cek log
docker compose logs -f
```

### Opsi B: Build lokal di STB
```bash
docker compose up -d --build
```

> **Untuk CasaOS / ZimaOS / Portainer di STB**:
> Anda cukup mengimpor file `docker-compose.yml` dan menyertakan file `.env`.

---

## 3. Menjalankan Tanpa Docker (Native Binary)

```bash
# Test cek 1x:
go run -buildvcs=false . --once

# Compile binary native STB (ARM64):
GOOS=linux GOARCH=arm64 go build -buildvcs=false -ldflags="-s -w" -o pokemon-monitor .
```
