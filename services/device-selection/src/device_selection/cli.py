from __future__ import annotations

from device_selection.core.model import Device, DeviceSelectionRequest, TypeCount, ParetoPoint
from device_selection.core.pareto import ObjectiveBounds
from device_selection.core.validate import validate_solution
from device_selection.data.catalog import Catalog, InMemoryCatalog
from device_selection.data.loader import load_catalog, type_id, eco_id
from device_selection.data.result_exporter import export_result
from device_selection.solvers.enum_repair import solve_enum_repair

import json
from pathlib import Path

_TEMPLATE = """<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Smart Home Optimizer — Results</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: system-ui, sans-serif; background: #f3f4f6; color: #1f2937; }
  header { background: #1e3a5f; color: #fff; padding: 1.2rem 2rem; }
  header h1 { font-size: 1.4rem; }
  .container { max-width: 1100px; margin: 2rem auto; padding: 0 1rem; }

  /* ── Pareto scatter ── */
  #chart-wrap { background: #fff; border-radius: 12px; padding: 1.5rem;
                box-shadow: 0 1px 6px #0001; margin-bottom: 2rem; }
  #chart-wrap h2 { margin-bottom: 1rem; font-size: 1rem; color: #6b7280; }

  /* ── Point cards ── */
  .point-card { background: #fff; border-radius: 12px; margin-bottom: 1.5rem;
                box-shadow: 0 1px 6px #0001; overflow: hidden; }
  .point-header { display: flex; align-items: center; gap: 1.5rem;
                  padding: 1rem 1.5rem; cursor: pointer;
                  border-bottom: 2px solid transparent; transition: background .15s; }
  .point-header:hover { background: #f9fafb; }
  .point-header.open { border-color: #3b82f6; }
  .badge { display: inline-flex; align-items: center; gap: .3rem;
           padding: .25rem .7rem; border-radius: 999px; font-size: .8rem; font-weight: 600; }
  .badge-cost   { background: #dbeafe; color: #1d4ed8; }
  .badge-qual   { background: #dcfce7; color: #15803d; }
  .badge-eco    { background: #fef9c3; color: #854d0e; }
  .badge-hub    { background: #fce7f3; color: #9d174d; }
  .point-title  { font-weight: 700; font-size: 1rem; flex: 1; }

  /* ── Device grid ── */
  .point-body { display: none; padding: 1.5rem; }
  .point-body.open { display: block; }
  .device-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 1rem; }
  .device-card { border: 1px solid #e5e7eb; border-radius: 10px; overflow: hidden;
                 display: flex; flex-direction: column; }
  .device-img  { width: 100%; height: 140px; object-fit: contain;
                 background: #f9fafb; padding: .5rem; }
  .device-img-placeholder { width: 100%; height: 140px; background: #f3f4f6;
                             display: flex; align-items: center; justify-content: center;
                             color: #9ca3af; font-size: .8rem; }
  .device-info { padding: .75rem; flex: 1; }
  .device-info h3 { font-size: .9rem; margin-bottom: .3rem; }
  .device-info .meta { font-size: .75rem; color: #6b7280; margin-bottom: .4rem; }
  .device-info .qty  { font-weight: 700; color: #2563eb; }
  .device-info .conn { font-size: .72rem; padding: .2rem .5rem; border-radius: 6px;
                       display: inline-block; margin-top: .4rem; }
  .conn-direct { background: #dcfce7; color: #166534; }
  .conn-bridge { background: #fef3c7; color: #92400e; }

  /* listings drawer */
  .listings-toggle { font-size: .72rem; color: #3b82f6; cursor: pointer;
                     padding: .4rem .75rem; border-top: 1px solid #f3f4f6; }
  .listings-drawer { display: none; padding: .5rem .75rem .75rem;
                     border-top: 1px solid #f3f4f6; }
  .listings-drawer.open { display: block; }
  .listing-row { font-size: .72rem; display: flex; justify-content: space-between;
                 gap: .5rem; padding: .25rem 0; border-bottom: 1px solid #f9fafb; }
  .listing-row a { color: #2563eb; text-decoration: none; flex: 1;
                   white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .listing-row a:hover { text-decoration: underline; }

  /* attrs drawer */
  .attrs-toggle { font-size: .72rem; color: #6b7280; cursor: pointer;
                  padding: .4rem .75rem; border-top: 1px solid #f3f4f6; }
  .attrs-drawer { display: none; padding: .5rem .75rem .75rem;
                  border-top: 1px solid #f3f4f6; }
  .attrs-drawer.open { display: block; }
  .attr-row { font-size: .72rem; display: flex; gap: .5rem; padding: .15rem 0; }
  .attr-key { color: #6b7280; min-width: 120px; }
  .attr-val { font-weight: 500; }
</style>
</head>
<body>
<header><h1>🏠 Smart Home Optimizer — Pareto Results</h1></header>
<div class="container">

<div id="chart-wrap">
  <h2>Pareto front: cost vs quality (bubble size = num ecosystems)</h2>
  <canvas id="chart" height="80"></canvas>
</div>

<div id="points"></div>

</div>

<script>
const DATA = __DATA__;

// ── render cards ─────────────────────────────────────────────────────────────
const container = document.getElementById("points");

DATA.pareto_points.forEach((pt, pi) => {
  const card = document.createElement("div");
  card.className = "point-card";

  const header = document.createElement("div");
  header.className = "point-header";
  header.innerHTML = `
    <span class="point-title">Bundle #${pi + 1}</span>
    <span class="badge badge-cost">💰 ${pt.total_cost.toLocaleString()} ₽</span>
    <span class="badge badge-qual">⭐ quality ${(pt.avg_quality * 5).toFixed(2)}</span>
    <span class="badge badge-eco">🔗 ${pt.num_ecosystems} ecosystem${pt.num_ecosystems !== 1 ? "s" : ""}</span>
    <span class="badge badge-hub">📡 ${pt.num_hubs} hub${pt.num_hubs !== 1 ? "s" : ""}</span>
  `;

  const body = document.createElement("div");
  body.className = "point-body";

  const grid = document.createElement("div");
  grid.className = "device-grid";

  pt.items.forEach(item => {
    const dc = document.createElement("div");
    dc.className = "device-card";

    const imgHtml = item.image_url
      ? `<img class="device-img" src="${item.image_url}" alt="" loading="lazy">`
      : `<div class="device-img-placeholder">no image</div>`;

    const connClass = item.connection.method === "direct" ? "conn-direct" : "conn-bridge";
    const connLabel = item.connection.method === "direct"
      ? "✅ direct"
      : `🔀 via ${item.connection.bridge_ecosystem || "bridge"}`;

    const attrsHtml = Object.entries(item.device_attributes || {})
      .filter(([, v]) => v !== null && v !== undefined)
      .map(([k, v]) => `
        <div class="attr-row">
          <span class="attr-key">${k}</span>
          <span class="attr-val">${Array.isArray(v) ? v.join(", ") : v}</span>
        </div>`).join("");

    const listingsHtml = (item.listings || []).map(l => `
      <div class="listing-row">
        <a href="${l.url || "#"}" target="_blank">${l.name || l.url || "listing"}</a>
        <span>${l.price != null ? l.price.toLocaleString() + " ₽" : "—"}</span>
        <span>⭐${l.rating ?? "—"}</span>
      </div>`).join("");

    dc.innerHTML = `
      ${imgHtml}
      <div class="device-info">
        <h3>${item.brand || ""} ${item.model || item.category}</h3>
        <div class="meta">${item.category}</div>
        <div>
          <span class="qty">× ${item.quantity}</span>
          &nbsp;
          <span style="font-size:.8rem">💰 ${item.price_each != null ? item.price_each.toLocaleString() + " ₽ each" : "—"}</span>
        </div>
        <div class="meta">⭐ quality ${(item.quality * 5).toFixed(2)}</div>
        <span class="conn ${connClass}">${connLabel}</span>
      </div>
      <div class="attrs-toggle" onclick="toggleDrawer(this)">▶ Attributes</div>
      <div class="attrs-drawer">${attrsHtml || "<span style='color:#9ca3af;font-size:.72rem'>none</span>"}</div>
      <div class="listings-toggle" onclick="toggleDrawer(this)">▶ ${(item.listings||[]).length} listing(s)</div>
      <div class="listings-drawer">${listingsHtml || "<span style='color:#9ca3af;font-size:.72rem'>none</span>"}</div>
    `;
    grid.appendChild(dc);
  });

  body.appendChild(grid);
  card.appendChild(header);
  card.appendChild(body);
  container.appendChild(card);

  header.addEventListener("click", () => {
    body.classList.toggle("open");
    header.classList.toggle("open");
  });

  // open first by default
  if (pi === 0) { body.classList.add("open"); header.classList.add("open"); }
});

// ── toggle helper ─────────────────────────────────────────────────────────────
function toggleDrawer(el) {
  const drawer = el.nextElementSibling;
  const open = drawer.classList.toggle("open");
  el.textContent = el.textContent.replace(/[▶▼]/, open ? "▼" : "▶");
}

// ── chart ─────────────────────────────────────────────────────────────────────
const pts = DATA.pareto_points;
const canvas = document.getElementById("chart");
const W = canvas.offsetWidth; const H = canvas.height = Math.max(200, W * 0.25);
const ctx = canvas.getContext("2d");
const pad = 50;

const costs    = pts.map(p => p.total_cost);
const quals    = pts.map(p => p.avg_quality);
const minC = Math.min(...costs), maxC = Math.max(...costs);
const minQ = Math.min(...quals), maxQ = Math.max(...quals);

const cx = c => pad + (maxC === minC ? W/2 : (c - minC) / (maxC - minC) * (W - 2*pad));
const cy = q => H - pad - (maxQ === minQ ? H/2 : (q - minQ) / (maxQ - minQ) * (H - 2*pad));

// axes
ctx.strokeStyle = "#d1d5db"; ctx.lineWidth = 1;
ctx.beginPath(); ctx.moveTo(pad, pad); ctx.lineTo(pad, H-pad); ctx.lineTo(W-pad, H-pad); ctx.stroke();
ctx.fillStyle = "#9ca3af"; ctx.font = "11px system-ui";
ctx.fillText("cost →", W - pad - 30, H - 10);
ctx.save(); ctx.translate(12, H/2); ctx.rotate(-Math.PI/2);
ctx.fillText("quality →", -25, 0); ctx.restore();

pts.forEach((p, i) => {
  const x = cx(p.total_cost), y = cy(p.avg_quality);
  const r = 6 + p.num_ecosystems * 3;
  ctx.beginPath(); ctx.arc(x, y, r, 0, Math.PI*2);
  ctx.fillStyle = "#3b82f680"; ctx.fill();
  ctx.strokeStyle = "#2563eb"; ctx.lineWidth = 1.5; ctx.stroke();
  ctx.fillStyle = "#1f2937"; ctx.font = "bold 11px system-ui";
  ctx.fillText(`#${i+1}`, x - 5, y - r - 4);
});
</script>
</body>
</html>
"""

def build_html(result_json: str | Path, output_html: str | Path) -> None:
    data = Path(result_json).read_text(encoding="utf-8")
    html = _TEMPLATE.replace("__DATA__", data)
    Path(output_html).write_text(html, encoding="utf-8")
    print(f"HTML written → {output_html}")


CATALOG_JSON  = "catalog.json"
RESULT_JSON   = "result.json"
RESULT_HTML   = "result.html"

# --- IDs (just integers) ---
MAIN = 1            # main ecosystem (e.g., Yandex)
TUYA = 2            # bridge ecosystem 1
XIAOMI = 3          # bridge ecosystem 2
IKEA   = 2001   # e.g., Ikea Home Smart ecosystem
HUE    = 2002   # e.g., Philips Hue ecosystem
LUTRON = 2003   # e.g., Lutron ecosystem (Caséta)

T_BULB = 10
T_MOTION = 11
T_LEAK = 12

# Hub device types (IMPORTANT: hub_type_id is also a device type id in your model)
T_HUB_ZIGBEE = 100
T_HUB_TUYA = 101

# New hub types (in addition to T_HUB_ZIGBEE, T_HUB_TUYA)
T_HUB_ZWAVE          = 3101
T_HUB_IKEA_TRADFRI   = 3102
T_HUB_HUE_BRIDGE     = 3103
T_HUB_LUTRON_CASETA  = 3104

def print_solution(idx: int, req: DeviceSelectionRequest, catalog: Catalog, p: ParetoPoint) -> None:
    print(f"\nSolution #{idx}")
    errors = validate_solution(req, p, catalog)
    if errors:
        print("Errors: ", errors)
    print(f"  total_cost={p.total_cost:.2f} avg_quality={p.avg_quality:.3f} "
          f"ecosystems={p.num_ecosystems} hubs={p.num_hubs}")
    print("  items:")
    for it in p.items:
        d = it.device
        conn = it.connection
        print(
            f"    - type={d.type_id} device_id={d.device_id} qty={it.quantity} "
            f"price={d.price:.2f} q={d.quality:.2f} "
            f"bridge={d.bridge_ecosystem_id} req_hub_type={d.hub_type_id} "
            f"plan=({conn.method.value}, bridge={conn.bridge_ecosystem_id}, hub_device_id={conn.hub_device_id})"
        )


def build_toy_catalog() -> InMemoryCatalog:
    """
    Catalog designed so that you plausibly get different non-dominated trade-offs:

    - "All direct Wi-Fi" -> ecosystems=1, hubs=0, quality lower
    - "Zigbee devices + Zigbee hub" -> ecosystems=1, hubs=1, quality higher, more cost
    - "Via Tuya devices + Tuya hub" -> ecosystems=2, hubs=1, quality maybe good/cheap but more ecosystems
    """

    dev_id = 1

    def D(
        type_id: int,
        price: float,
        quality: float,
        bridge: int | None,
        hub_type: int | None,
    ) -> Device:
        nonlocal dev_id
        d = Device(
            device_id=dev_id,
            type_id=type_id,
            price=price,
            quality=quality,
            bridge_ecosystem_id=bridge,
            hub_type_id=hub_type,
        )
        dev_id += 1
        return d

    devices_by_type: dict[int, list[Device]] = {
        # --- Light bulbs (3 needed) ---
        T_BULB: [
            # Direct Wi-Fi bulbs (no hubs, no bridge)
            D(T_BULB, price=8,  quality=0.55, bridge=None, hub_type=None),
            D(T_BULB, price=12, quality=0.70, bridge=None, hub_type=None),

            # Zigbee bulbs (direct to MAIN but require Zigbee hub)
            D(T_BULB, price=6,  quality=0.60, bridge=None, hub_type=T_HUB_ZIGBEE),
            D(T_BULB, price=10, quality=0.80, bridge=None, hub_type=T_HUB_ZIGBEE),

            # Bulbs only available via Tuya (bridge ecosystem TUYA); no extra hub
            D(T_BULB, price=7,  quality=0.75, bridge=TUYA, hub_type=None),
            D(T_BULB, price=11, quality=0.90, bridge=TUYA, hub_type=None),

            # Hue bulbs (bridge ecosystem HUE) require Hue Bridge hub
            D(T_BULB, price=13, quality=0.82, bridge=HUE, hub_type=T_HUB_HUE_BRIDGE),
            D(T_BULB, price=22, quality=0.94, bridge=HUE, hub_type=T_HUB_HUE_BRIDGE),

            # IKEA bulbs (bridge ecosystem IKEA) require Tradfri hub
            D(T_BULB, price=9,  quality=0.73, bridge=IKEA, hub_type=T_HUB_IKEA_TRADFRI),
            D(T_BULB, price=14, quality=0.84, bridge=IKEA, hub_type=T_HUB_IKEA_TRADFRI),
        ],

        # --- Motion sensors (3 needed) ---
        T_MOTION: [
            # Direct Wi-Fi motion sensors
            D(T_MOTION, price=9,  quality=0.60, bridge=None, hub_type=None),
            D(T_MOTION, price=14, quality=0.78, bridge=None, hub_type=None),

            # Zigbee motion sensors (require Zigbee hub)
            D(T_MOTION, price=7,  quality=0.70, bridge=None, hub_type=T_HUB_ZIGBEE),
            D(T_MOTION, price=11, quality=0.88, bridge=None, hub_type=T_HUB_ZIGBEE),

            # Tuya motion sensors (require Tuya hub AND bridge TUYA)
            D(T_MOTION, price=6,  quality=0.72, bridge=TUYA, hub_type=T_HUB_TUYA),
            D(T_MOTION, price=10, quality=0.86, bridge=TUYA, hub_type=T_HUB_TUYA),

            # Z-Wave motion sensors (direct to MAIN, require Z-Wave hub)
            D(T_MOTION, price=12, quality=0.83, bridge=None, hub_type=T_HUB_ZWAVE),
            D(T_MOTION, price=18, quality=0.91, bridge=None, hub_type=T_HUB_ZWAVE),

            # IKEA motion sensors (bridge IKEA, require Tradfri hub)
            D(T_MOTION, price=8,  quality=0.74, bridge=IKEA, hub_type=T_HUB_IKEA_TRADFRI),
            D(T_MOTION, price=13, quality=0.85, bridge=IKEA, hub_type=T_HUB_IKEA_TRADFRI),
        ],

        # --- Water leak sensors (2 needed) ---
        T_LEAK: [
            # Zigbee leak sensors
            D(T_LEAK, price=8,  quality=0.72, bridge=None, hub_type=T_HUB_ZIGBEE),
            D(T_LEAK, price=12, quality=0.90, bridge=None, hub_type=T_HUB_ZIGBEE),

            # Alternative via Tuya (bridge TUYA + Tuya hub)
            D(T_LEAK, price=7,  quality=0.74, bridge=TUYA, hub_type=T_HUB_TUYA),
            D(T_LEAK, price=11, quality=0.87, bridge=TUYA, hub_type=T_HUB_TUYA),

            # Z-Wave leak sensors (require Z-Wave hub)
            D(T_LEAK, price=10, quality=0.82, bridge=None, hub_type=T_HUB_ZWAVE),
            D(T_LEAK, price=16, quality=0.93, bridge=None, hub_type=T_HUB_ZWAVE),

            # IKEA leak sensor (bridge IKEA + Tradfri hub) if you want the option
            D(T_LEAK, price=9,  quality=0.78, bridge=IKEA, hub_type=T_HUB_IKEA_TRADFRI),
        ],

        # --- Zigbee hub devices ---
        T_HUB_ZIGBEE: [
            D(T_HUB_ZIGBEE, price=20, quality=0.60, bridge=None, hub_type=None),
            D(T_HUB_ZIGBEE, price=35, quality=0.85, bridge=None, hub_type=None),
        ],

        # --- Tuya hub devices ---
        T_HUB_TUYA: [
            D(T_HUB_TUYA, price=15, quality=0.55, bridge=TUYA, hub_type=None),
            D(T_HUB_TUYA, price=28, quality=0.80, bridge=TUYA, hub_type=None),
        ],

        # --- Z-Wave hub devices ---
        T_HUB_ZWAVE: [
            D(T_HUB_ZWAVE, price=40, quality=0.70, bridge=None, hub_type=None),
            D(T_HUB_ZWAVE, price=65, quality=0.90, bridge=None, hub_type=None),
        ],

        # --- IKEA Tradfri hub devices (bridged via IKEA ecosystem) ---
        T_HUB_IKEA_TRADFRI: [
            D(T_HUB_IKEA_TRADFRI, price=30, quality=0.65, bridge=IKEA, hub_type=None),
            D(T_HUB_IKEA_TRADFRI, price=45, quality=0.82, bridge=IKEA, hub_type=None),
        ],

        # --- Hue Bridge hub devices (bridged via HUE ecosystem) ---
        T_HUB_HUE_BRIDGE: [
            D(T_HUB_HUE_BRIDGE, price=45, quality=0.75, bridge=HUE, hub_type=None),
            D(T_HUB_HUE_BRIDGE, price=60, quality=0.88, bridge=HUE, hub_type=None),
        ],

        # --- Lutron Caséta hub devices (bridged via LUTRON ecosystem) ---
        T_HUB_LUTRON_CASETA: [
            D(T_HUB_LUTRON_CASETA, price=70, quality=0.80, bridge=LUTRON, hub_type=None),
            D(T_HUB_LUTRON_CASETA, price=95, quality=0.92, bridge=LUTRON, hub_type=None),
        ],
    }
    return InMemoryCatalog(devices_by_type)


def run_once(budget: float) -> None:
    catalog, enriched_devices = load_catalog(CATALOG_JSON)

    # build ids AFTER loading so registries are populated
    T_BULB   = type_id("smart_lamp")
    T_MOTION = type_id("motion_sensor")
    T_LEAK   = type_id("water_leak_sensor")
    T_DOOR = type_id("door_window_sensor")
    MAIN     = eco_id("yandex")

    req = DeviceSelectionRequest(
        main_ecosystem_id = MAIN,
        budget            = budget,
        type_counts       = (
            TypeCount(type_id=T_BULB,   count=3),
            TypeCount(type_id=T_MOTION, count=2),
            TypeCount(type_id=T_LEAK,   count=5),
            TypeCount(type_id=T_DOOR,   count=3),
        ),
        max_solutions          = 7,
        random_seed            = 123,
        time_budget_seconds    = 5.0,
    )

    archive = solve_enum_repair(req, catalog)
    export_result(archive.points, enriched_devices, RESULT_JSON)
    build_html(RESULT_JSON, RESULT_HTML)   # see below
    print(f"HTML → {RESULT_HTML}")

    archive = solve_enum_repair(req, catalog)
    points = archive.points

    print(f"\n=== Budget = {budget:.2f} | Found {len(points)} pareto points ===")
    print("HV: ", archive.hypervolume(ObjectiveBounds()))
    print("Ranges: ", archive.objective_ranges())
    print("Diversity: ", archive.diversity_avg_distance(ObjectiveBounds()))
    for i, p in enumerate(points, start=1):
        print_solution(i, req, catalog, p)


def main() -> None:
    # Try multiple budgets to see different tradeoffs show up
    run_once(40000.0)


if __name__ == "__main__":
    main()
