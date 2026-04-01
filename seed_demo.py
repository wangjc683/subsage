#!/usr/bin/env python3
import urllib.request, json, sys

BASE = "http://localhost:8321"

# Login
data = json.dumps({"username":"demo","password":"demo123"}).encode()
req = urllib.request.Request(f"{BASE}/api/auth/login", data=data, headers={"Content-Type":"application/json"})
token = json.loads(urllib.request.urlopen(req, timeout=5).read())["token"]
print(f"Logged in OK")

headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

subs = [
    {"name":"ChatGPT Plus","category":"ai","price":20,"currency":"USD","cycle":"monthly","status":"active","start_date":"2025-09-15","next_renewal":"2026-04-15","payment_method":"Visa *4242"},
    {"name":"Claude Pro","category":"ai","price":20,"currency":"USD","cycle":"monthly","status":"active","start_date":"2025-11-01","next_renewal":"2026-04-02","payment_method":"Visa *4242"},
    {"name":"Midjourney","category":"ai","price":30,"currency":"USD","cycle":"monthly","status":"active","start_date":"2025-08-01","next_renewal":"2026-04-08","payment_method":"Visa *4242"},
    {"name":"Cursor Pro","category":"dev","price":20,"currency":"USD","cycle":"monthly","status":"active","start_date":"2026-01-10","next_renewal":"2026-04-10","payment_method":"Visa *4242"},
    {"name":"GitHub Copilot","category":"dev","price":0,"original_price":10,"currency":"USD","cycle":"monthly","status":"active","start_date":"2025-06-01","next_renewal":"2026-04-01","discount_note":"GitHub Student Pack"},
    {"name":"Spotify Premium","category":"music","price":11.99,"currency":"USD","cycle":"monthly","status":"active","start_date":"2024-03-20","next_renewal":"2026-04-20","payment_method":"Apple Pay"},
    {"name":"YouTube Premium","category":"video","price":13.99,"currency":"USD","cycle":"monthly","status":"active","start_date":"2024-08-01","next_renewal":"2026-04-05","payment_method":"Google Pay"},
    {"name":"Netflix","category":"video","price":15.49,"currency":"USD","cycle":"monthly","status":"cancelled","start_date":"2023-01-15"},
    {"name":"iCloud+ 200GB","category":"cloud","price":2.99,"currency":"USD","cycle":"monthly","status":"active","start_date":"2023-06-01","next_renewal":"2026-04-01","payment_method":"Apple Pay"},
    {"name":"Cloudflare Pro","category":"cloud","price":25,"currency":"USD","cycle":"monthly","status":"active","start_date":"2025-03-01","next_renewal":"2026-04-12"},
    {"name":"1Password","category":"security","price":35.88,"currency":"USD","cycle":"yearly","status":"active","start_date":"2025-07-15","next_renewal":"2026-07-15","payment_method":"Visa *4242"},
    {"name":"Notion Plus","category":"software","price":10,"currency":"USD","cycle":"monthly","status":"active","start_date":"2025-02-01","next_renewal":"2026-04-03","payment_method":"Visa *4242"},
    {"name":"Adobe Creative Cloud","category":"software","price":54.99,"currency":"USD","cycle":"monthly","status":"paused","start_date":"2024-01-01"},
    {"name":"Nintendo Switch Online","category":"gaming","price":19.99,"currency":"USD","cycle":"yearly","status":"active","start_date":"2025-12-25","next_renewal":"2026-12-25"},
    {"name":"Amazon Prime","category":"membership","price":139,"currency":"USD","cycle":"yearly","status":"active","start_date":"2025-11-20","next_renewal":"2026-11-20"},
]

for s in subs:
    req = urllib.request.Request(f"{BASE}/api/subs", json.dumps(s).encode(), headers=headers, method="POST")
    try:
        urllib.request.urlopen(req, timeout=5)
        print(f"  ✅ {s['name']}")
    except Exception as e:
        print(f"  ❌ {s['name']}: {e}")

print(f"\nDone! Added {len(subs)} demo subscriptions.")
