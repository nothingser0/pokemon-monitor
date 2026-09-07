# pokemon-monitor

Pantau koordinat Pokemon dari `coordinates-api.pokemongopro.com`, kirim notifikasi ke Discord.

## Yang dibutuhkan

- Docker + Docker Compose (kalau pakai CasaOS sudah ada), atau
- Python 3.7+ dengan `pip install -r requirements.txt`

## Setup

Salin `.env.example` jadi `.env`, lalu isi:

```bash
cp .env.example .env
```

Isi `.env`:

```env
WEBHOOK_URL=https://discord.com/api/webhooks/...
BEARER_TOKEN=...
```

`config.json` berisi filter pencarian (daftar `pokemon_ids`, `species_forms`, IV/CP/level, dsb). Ini bukan kredensial, aman.

Default (`pokemon_ids: []`, `species_forms: []`) artinya ambil semua pokemon. Kalau mau filter, salin `config.json.example` ke `config.json` lalu isi daftarnya. Daftar favorit tersimpan di `favorites.json` sebagai referensi.

`config.json.example` memakai JSONC (ada komentar `//`) sebagai dokumentasi nilai yang valid (opsi `sort`, `weather_id`, dll). Kalau dipakai sebagai `config.json`, hapus dulu semua komentarnya — `config.json` harus JSON murni.

## Jalankan

### Docker

```bash
docker compose up -d --build
```

Lihat log:

```bash
docker compose logs -f
```

Stop:

```bash
docker compose down
```

Untuk CasaOS: App Store → Custom Install → import `docker-compose.yml`.

State (daftar Pokemon yang sudah dinotifikasi) disimpan di folder `./data`. Kalau folder itu dibuat sebagai root, beri akses dulu:

```bash
mkdir -p data && sudo chown -R 1000:1000 data
```

### Tanpa Docker

Jalan sekali:

```bash
python3 monitor.py --once
```

Jalan terus di background:

```bash
./run.sh
```

Lihat log: `tail -f monitor.log`
Stop: `kill $(cat .monitor.pid)`

### systemd

```bash
sudo ./setup-autostart.sh
```

## Cara kerja

Script poll API, bandingkan hasil dengan `.state.json` (biar Pokemon yang sama tidak dinotifikasi dua kali), kirim embed ke Discord, lalu tidur dengan interval acak (dari `interval_min` sampai `interval_max` di `config.json`).

## File

- `monitor.py` — script utama
- `config.json` — filter pencarian (aktif)
- `config.json.example` — template filter
- `favorites.json` — daftar pokemon favorit (referensi)
- `.env` — kredensial (jangan di-commit)
- `Dockerfile`, `docker-compose.yml` — deployment Docker
- `run.sh`, `setup-autostart.sh`, `pokemon-monitor.service` — deployment tanpa Docker
