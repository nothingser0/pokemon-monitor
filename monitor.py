#!/usr/bin/env python3
"""
Pokemon GO Coordinate Monitor

Usage:
  python3 monitor.py           # Run monitoring (loops forever)
  python3 monitor.py --once    # Run once and exit
  python3 monitor.py --config  # Show config paths
"""

import json
import os
import random
import socket
import sys
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path
from urllib.parse import urlparse

import requests

try:
    from dotenv import load_dotenv
except ImportError:  # pragma: no cover - optional dependency fallback
    load_dotenv = None

SCRIPT_DIR = Path(__file__).parent
DEFAULT_ENV_FILE = SCRIPT_DIR / ".env"


def load_env():
    if load_dotenv is not None:
        load_dotenv(DEFAULT_ENV_FILE)
    elif DEFAULT_ENV_FILE.exists():
        for line in DEFAULT_ENV_FILE.read_text().splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            key = key.strip()
            value = value.strip().strip('"').strip("'")
            if key and key not in os.environ:
                os.environ[key] = value


def _path_from_env(env_name, default):
    raw = os.environ.get(env_name)
    if raw:
        return Path(raw)
    return SCRIPT_DIR / default


load_env()

CONFIG_FILE = _path_from_env("CONFIG_FILE", "config.json")
ENV_FILE = _path_from_env("ENV_FILE", ".env")
STATE_FILE = _path_from_env("STATE_FILE", ".state.json")
MAX_EMBEDS_PER_BATCH = 10


def get_env(key, default=None):
    value = os.environ.get(key)
    if value is None or value == "":
        return default
    return value


def load_config():
    if CONFIG_FILE.exists():
        try:
            return json.loads(CONFIG_FILE.read_text())
        except Exception as e:
            print(f"⚠️  Config parse error: {e}, using default", file=sys.stderr)
            return build_default_config()
    else:
        config = build_default_config()
        CONFIG_FILE.write_text(json.dumps(config, indent=2))
        print(f"✅ Created default config: {CONFIG_FILE}")
        return config


def build_default_config():
    return {
        "api_url": "https://coordinates-api.pokemongopro.com/search",
        "interval_min": 10,
        "interval_max": 60,
        "payload": {
            "pvp_mode": False,
            "pvp_league": "great",
            "pvp_top_rank": None,
            "pvp_include_pre_evolutions": True,
            "pvp_min_cp": None,
            "pokemon_ids": [],
            "species_forms": [],
            "location": None,
            "cp_min": None,
            "cp_max": None,
            "level_min": None,
            "level_max": None,
            "iv_mode": "percent",
            "iv_percent_min": 0,
            "iv_percent_max": None,
            "raw_iv_atk": None,
            "raw_iv_def": None,
            "raw_iv_sta": None,
            "sizes": [],
            "forms": [],
            "gender": None,
            "boosted": None,
            "weather_id": None,
            "min_ttl": None,
            "sort": "IV",
            "center_lat": None,
            "center_lng": None,
            "radius_km": None,
            "blacklist_pokemon_ids": [],
            "blacklist_species_forms": [],
            "blacklist_forms": [],
        },
    }


def load_state():
    if STATE_FILE.exists():
        try:
            return set(json.loads(STATE_FILE.read_text()))
        except Exception:
            return set()
    return set()


def save_state(notified_ids):
    STATE_FILE.write_text(json.dumps(list(notified_ids)))


def fetch_pokemon(api_url, bearer_token, payload):
    headers = {
        "Authorization": f"Bearer {bearer_token}",
        "Content-Type": "application/json",
    }
    
    parsed = urlparse(api_url)
    hostname = parsed.hostname
    fallback_ip = "46.225.53.252"
    
    try:
        socket.gethostbyname(hostname)
        target_url = api_url
    except (socket.gaierror, socket.herror, OSError):
        print(f"⚠️  DNS resolution failed for {hostname}, using fallback IP {fallback_ip}", file=sys.stderr)
        target_url = api_url.replace(hostname, fallback_ip)
        headers["Host"] = hostname
    
    resp = requests.post(target_url, json=payload, headers=headers, timeout=30)
    resp.raise_for_status()
    return resp.json()


def parse_despawn(despawn_str):
    if not despawn_str:
        return None

    try:
        return datetime.fromisoformat(despawn_str)
    except ValueError:
        pass

    if despawn_str.endswith("Z"):
        try:
            return datetime.fromisoformat(despawn_str[:-1]).replace(tzinfo=timezone.utc)
        except ValueError:
            pass

    try:
        from dateutil import parser
        return parser.parse(despawn_str)
    except Exception:
        return None


def build_countdown(despawn_str):
    despawn_dt = parse_despawn(despawn_str)
    if not despawn_dt:
        return "Unknown"

    if despawn_dt.tzinfo is None:
        despawn_dt = despawn_dt.replace(tzinfo=timezone.utc)
    unix_ts = int(despawn_dt.timestamp())
    return f"<t:{unix_ts}:R>"


def build_embed(p):
    pokemon_name = p.get("pokemon_name", "Unknown")
    pokemon_id = p.get("pokemon_id", "?")
    gender = p.get("gender", "Unknown")
    cp = p.get("cp", 0)
    level = p.get("level", 0)
    iv = p.get("iv_percent", 0)
    raw_iv = p.get("raw_iv", "?/?/?")
    location = p.get("location", "Unknown")
    reveal_code = p.get("reveal_code", "")
    weather = p.get("weather_boosted", False)
    fast_move = p.get("fast_move", "Unknown")
    charge_move = p.get("charge_move", "Unknown")

    if reveal_code:
        reveal_url = f"https://coordinates-api.pokemongopro.com/reveal/{reveal_code}"
    else:
        maps_query = location.replace(" ", "+").replace(",", "%2C")
        reveal_url = f"https://www.google.com/maps/search/?api=1&query={maps_query}"

    countdown_text = build_countdown(p.get("despawn_at"))

    gender_emoji = "♂️" if gender == "Male" else "♀️" if gender == "Female" else ""
    weather_emoji = " ☀️" if weather else ""

    if iv >= 90:
        color = 0x00FF00
    elif iv >= 80:
        color = 0xFFAA00
    else:
        color = 0x3498DB

    image_url = (
        "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/"
        f"pokemon/other/official-artwork/{pokemon_id}.png"
    )

    return {
        "title": f"{pokemon_name} {gender_emoji}{weather_emoji}",
        "color": color,
        "thumbnail": {"url": image_url},
        "fields": [
            {"name": "IV", "value": f"**{iv}%** ({raw_iv})", "inline": True},
            {"name": "Level", "value": f"**{level}**", "inline": True},
            {"name": "CP", "value": f"**{cp}**", "inline": True},
            {"name": "Moves", "value": f"{fast_move} / {charge_move}", "inline": False},
            {"name": "Despawns", "value": countdown_text, "inline": False},
            {"name": "Location", "value": f"[{location}]({reveal_url})", "inline": False},
        ],
        "footer": {"text": f"ID: {pokemon_id}"},
    }


def send_discord(webhook_url, pokemon_list):
    if not pokemon_list:
        return

    batch = pokemon_list[:MAX_EMBEDS_PER_BATCH]
    for p in batch:
        embed = build_embed(p)
        try:
            resp = requests.post(webhook_url, json={"embeds": [embed]}, timeout=10)
            resp.raise_for_status()
            print(
                f"✅ Notified: {p.get('pokemon_name', 'Unknown')} "
                f"(#{p.get('pokemon_id', '?')}) at {p.get('location', 'Unknown')}"
            )
        except Exception as e:
            print(
                f"❌ Discord webhook error for {p.get('pokemon_name', 'Unknown')}: {e}",
                file=sys.stderr,
            )

    remaining = len(pokemon_list) - len(batch)
    if remaining > 0:
        try:
            requests.post(
                webhook_url,
                json={"content": f"...{remaining} lainnya"},
                timeout=10,
            )
        except Exception:
            pass


def random_interval(config):
    min_minutes = int(config.get("interval_min", 10))
    max_minutes = int(config.get("interval_max", 60))
    if min_minutes > max_minutes:
        min_minutes, max_minutes = max_minutes, min_minutes
    minutes = random.randint(min_minutes, max_minutes)
    return minutes * 60


def check_once(config, webhook_url, bearer_token):
    print(f"🔍 Checking Pokemon API at {datetime.now()}")

    payload = config.get("payload", {})

    try:
        data = fetch_pokemon(config.get("api_url"), bearer_token, payload)
    except Exception as e:
        print(f"❌ API error: {e}", file=sys.stderr)
        return

    results = data.get("results", [])
    total_matches = data.get("total_matches", 0)
    tier = data.get("tier", "unknown")

    print(f"📊 Tier: {tier} | Matches: {total_matches} | Results: {len(results)}")

    if not results:
        print("✅ No Pokemon found")
        return

    notified = load_state()
    new_pokemon = []
    for p in results:
        uid = p.get("encounter_id") or f"{p.get('pokemon_id')}_{p.get('lat')}_{p.get('lng')}"
        if uid not in notified:
            new_pokemon.append(p)
            notified.add(uid)

    if new_pokemon:
        send_discord(webhook_url, new_pokemon)
        save_state(notified)
        print(f"✅ {len(new_pokemon)} new Pokemon detected and notified")
    else:
        print("ℹ️  All Pokemon already notified")


def main():
    if len(sys.argv) > 1 and sys.argv[1] == "--config":
        print(f"📝 Config file: {CONFIG_FILE}")
        print(f"📝 Env file: {ENV_FILE}")
        print(f"📝 State file: {STATE_FILE}")
        print(f"\nEdit credentials: nano {ENV_FILE}")
        print(f"Edit filters:     nano {CONFIG_FILE}")
        return

    run_once = "--once" in sys.argv

    webhook_url = get_env("WEBHOOK_URL")
    bearer_token = get_env("BEARER_TOKEN")

    if not webhook_url or not bearer_token:
        print("❌ Missing configuration. Please set WEBHOOK_URL and BEARER_TOKEN", file=sys.stderr)
        print(f"   Edit {ENV_FILE} (see .env.example for reference)", file=sys.stderr)
        sys.exit(1)

    config = load_config()

    if run_once:
        check_once(config, webhook_url, bearer_token)
        return

    print("🔄 Starting continuous monitoring (Ctrl+C to stop)")
    while True:
        try:
            check_once(config, webhook_url, bearer_token)
        except KeyboardInterrupt:
            print("\n👋 Stopped by user")
            break
        except Exception as e:
            print(f"❌ Unexpected error: {e}", file=sys.stderr)

        interval = random_interval(config)
        print(f"--- Next check in {interval // 60} minutes ---")
        try:
            time.sleep(interval)
        except KeyboardInterrupt:
            print("\n👋 Stopped by user")
            break


if __name__ == "__main__":
    main()
