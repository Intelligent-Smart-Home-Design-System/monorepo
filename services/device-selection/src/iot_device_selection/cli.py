from __future__ import annotations

from iot_device_selection.core.model import Device, DeviceSelectionRequest, TypeCount, ParetoPoint
from iot_device_selection.core.pareto import ObjectiveBounds
from iot_device_selection.core.validate import validate_solution
from iot_device_selection.data.catalog import Catalog, InMemoryCatalog
from iot_device_selection.solvers.enum_repair import solve_enum_repair


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
    catalog = build_toy_catalog()

    req = DeviceSelectionRequest(
        main_ecosystem_id=MAIN,
        budget=budget,
        type_counts=(
            TypeCount(type_id=T_BULB, count=3),
            TypeCount(type_id=T_MOTION, count=3),
            TypeCount(type_id=T_LEAK, count=2),
        ),
        # You can try restricting bridges:
        # include_ecosystem_ids=frozenset({TUYA}),
        exclude_ecosystem_ids=frozenset(),
        max_solutions=7,
        random_seed=123,
        time_budget_seconds=2.0,   # small for toy example; increase to see more
    )

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
    run_once(60.0)
    run_once(90.0)
    run_once(120.0)
    run_once(1000.0)


if __name__ == "__main__":
    main()
