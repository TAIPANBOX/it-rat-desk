import json, sys, time, urllib.request
Q = [
 "Which of your services would fit a company that runs a dozen agents?",
 "Як встановити весь стек на один сервер?",
 "How do I reach a person on your team?",
]
for q in Q:
    body = json.dumps({"provider": "haiku", "history": [{"role": "user", "text": q}]}).encode()
    req = urllib.request.Request("http://127.0.0.1:8131/api/ask", body, {"Content-Type": "application/json"})
    t0 = time.time(); first = {}; deltas = 0; resets = 0; done = None
    with urllib.request.urlopen(req, timeout=120) as r:
        for raw in r:
            line = raw.decode().strip()
            if not line: continue
            e = json.loads(line); k = e["t"]
            first.setdefault(k, round(time.time() - t0, 2))
            deltas += k == "delta"; resets += k == "reset"
            if k == "done": done = e["done"]
    print(f"Q: {q[:60]}")
    print(f"   status at {first.get('status','-')}s, first word at {first.get('delta','-')}s, done at {first.get('done')}s | deltas={deltas} resets={resets} server first_ms={done.get('first_ms')} cost=${done.get('cost_usd'):.4f} pages={len(done.get('pages_read') or [])} contact={done.get('offer_contact')} err={done.get('error')!r}")
    print("   A:", done.get("answer","")[:160].replace("\n"," "))
