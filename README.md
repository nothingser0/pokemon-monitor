# pokemon-monitor

Pantau koordinat Pokemon dari `coordinates-api.pokemongopro.com`, kirim notifikasi ke Discord.

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

`config.json` berisi filter pencarian (pokemon_ids, species_forms, IV/CP/level). Default (kosong) = ambil semua pokemon.

## Jalankan

### Docker

```bash
docker compose up -d --build
docker compose logs -f
docker compose down
```

State (daftar Pokemon yang sudah dinotifikasi) disimpan di folder `./data`. Kalau folder dibuat sebagai root:

```bash
mkdir -p data && sudo chown -R 1000:1000 data
```

### Tanpa Docker

```bash
python3 monitor.py --once              # Run once
./run.sh                               # Background
tail -f monitor.log                    # Logs
kill $(cat .monitor.pid)               # Stop
```

### systemd

```bash
sudo ./setup-autostart.sh
```

## File

- `monitor.py` — script utama
- `config.json` — filter pencarian
- `config.json.example` — template
- `favorites.json` — referensi pokemon
- `.env` — kredensial (jangan commit)
- `Dockerfile`, `docker-compose.yml` — Docker deployment
- `run.sh`, `setup-autostart.sh`, `pokemon-monitor.service` — systemd deployment
