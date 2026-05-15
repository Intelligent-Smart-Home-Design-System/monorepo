from __future__ import annotations

import pytest

from device_selection.core.model import (
    BridgeCompat,
    ConnectionPlan,
    Device,
    DirectCompat,
    HubType,
)
from device_selection.core.pathfinding import find_connection


def make_device(
    device_id: int = 1,
    device_type: str = "smart_lamp",
    direct_compat: tuple[DirectCompat, ...] = (),
    bridge_compat: tuple[BridgeCompat, ...] = (),
) -> Device:
    return Device(
        device_id=device_id,
        device_type=device_type,
        brand="test",
        model="test",
        attributes={},
        price=1000.0,
        quality=0.8,
        source_listing_id=1,
        direct_compat=direct_compat,
        bridge_compat=bridge_compat,
    )


MAIN = "google"
ALL_ECOSYSTEMS = frozenset({"google", "aqara", "tuya", "yandex"})
NO_HUBS: frozenset[HubType] = frozenset()

# hub types used across tests
AQARA_ZIGBEE_HUB = HubType(ecosystem="aqara", protocols=frozenset({"zigbee"}))
AQARA_ZIGBEE_MATTER_THREAD_HUB = HubType(
    ecosystem="aqara", protocols=frozenset({"zigbee", "matter-over-thread"})
)
GOOGLE_MATTER_THREAD_HUB = HubType(
    ecosystem="google", protocols=frozenset({"matter-over-thread"})
)
GOOGLE_ZIGBEE_HUB = HubType(ecosystem="google", protocols=frozenset({"zigbee"}))
TUYA_ZIGBEE_HUB = HubType(ecosystem="tuya", protocols=frozenset({"zigbee"}))


def _direct(plan: ConnectionPlan):
    return plan.connection_direct


def _final(plan: ConnectionPlan):
    return plan.connection_final


class TestDirectToMain:
    def test_wifi_direct_to_main_no_hub_needed(self):
        device = make_device(direct_compat=(DirectCompat("google", "wifi"),))
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is not None
        assert _direct(plan).ecosystem == "google"
        assert _direct(plan).protocol == "wifi"
        assert _final(plan) is None

    def test_zigbee_direct_to_main_hub_available(self):
        device = make_device(direct_compat=(DirectCompat("google", "zigbee"),))
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, frozenset({GOOGLE_ZIGBEE_HUB}))
        assert plan is not None
        assert _direct(plan).ecosystem == "google"
        assert _direct(plan).protocol == "zigbee"
        assert _final(plan) is None

    def test_zigbee_direct_to_main_no_hub_returns_none(self):
        device = make_device(direct_compat=(DirectCompat("google", "zigbee"),))
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is None

    def test_zigbee_direct_to_main_wrong_ecosystem_hub_returns_none(self):
        # hub exists but for aqara not google
        device = make_device(direct_compat=(DirectCompat("google", "zigbee"),))
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, frozenset({AQARA_ZIGBEE_HUB}))
        assert plan is None

    def test_direct_main_ecosystem_not_in_available(self):
        device = make_device(direct_compat=(DirectCompat("google", "wifi"),))
        plan = find_connection(device, True, MAIN, frozenset({"aqara", "tuya"}), NO_HUBS)
        assert plan is None

    def test_direct_preferred_over_bridge_when_both_possible(self):
        device = make_device(
            direct_compat=(
                DirectCompat("google", "wifi"),
                DirectCompat("aqara", "wifi"),
            ),
            bridge_compat=(BridgeCompat("aqara", "google", "cloud"),),
        )
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is not None
        assert _direct(plan).ecosystem == "google"
        assert _final(plan) is None

    def test_multiple_direct_compat_picks_main(self):
        device = make_device(
            direct_compat=(
                DirectCompat("aqara", "wifi"),
                DirectCompat("google", "wifi"),
                DirectCompat("tuya", "wifi"),
            ),
        )
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is not None
        assert _direct(plan).ecosystem == "google"

    def test_no_compat_records_returns_none(self):
        device = make_device()
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is None


class TestCloudBridge:
    def test_cloud_bridge_wifi_source_no_hubs_needed(self):
        device = make_device(
            direct_compat=(DirectCompat("aqara", "wifi"),),
            bridge_compat=(BridgeCompat("aqara", "google", "cloud"),),
        )
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is not None
        assert _direct(plan).ecosystem == "aqara"
        assert _direct(plan).protocol == "wifi"
        assert _final(plan) is not None
        assert _final(plan).ecosystem == "google"
        assert _final(plan).protocol == "cloud"

    def test_cloud_bridge_zigbee_source_hub_available(self):
        device = make_device(
            direct_compat=(DirectCompat("aqara", "zigbee"),),
            bridge_compat=(BridgeCompat("aqara", "google", "cloud"),),
        )
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, frozenset({AQARA_ZIGBEE_HUB}))
        assert plan is not None
        assert _direct(plan).ecosystem == "aqara"
        assert _direct(plan).protocol == "zigbee"
        assert _final(plan).ecosystem == "google"
        assert _final(plan).protocol == "cloud"

    def test_cloud_bridge_zigbee_source_no_hub_returns_none(self):
        device = make_device(
            direct_compat=(DirectCompat("aqara", "zigbee"),),
            bridge_compat=(BridgeCompat("aqara", "google", "cloud"),),
        )
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is None

    def test_cloud_bridge_source_ecosystem_not_available(self):
        device = make_device(
            direct_compat=(DirectCompat("aqara", "wifi"),),
            bridge_compat=(BridgeCompat("aqara", "google", "cloud"),),
        )
        # aqara not in available ecosystems
        plan = find_connection(device, True, MAIN, frozenset({"google", "tuya"}), NO_HUBS)
        assert plan is None

    def test_cloud_bridge_wrong_target_returns_none(self):
        device = make_device(
            direct_compat=(DirectCompat("aqara", "wifi"),),
            bridge_compat=(BridgeCompat("aqara", "yandex", "cloud"),),
        )
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is None

    def test_cloud_bridge_no_direct_compat_for_source_returns_none(self):
        # bridge record says aqara->google but device has no direct compat for aqara
        device = make_device(
            direct_compat=(DirectCompat("tuya", "wifi"),),
            bridge_compat=(BridgeCompat("aqara", "google", "cloud"),),
        )
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is None


class TestMatterBridge:
    def test_matter_thread_bridge_both_hubs_available(self):
        # device connects to aqara via zigbee, aqara bridges to google via matter-over-thread
        # requires: aqara hub supporting zigbee+matter-over-thread, google hub supporting matter-over-thread
        device = make_device(
            direct_compat=(DirectCompat("aqara", "zigbee"),),
            bridge_compat=(BridgeCompat("aqara", "google", "matter-over-thread"),),
        )
        plan = find_connection(
            device, True, MAIN, ALL_ECOSYSTEMS,
            frozenset({AQARA_ZIGBEE_MATTER_THREAD_HUB, GOOGLE_MATTER_THREAD_HUB}),
        )
        assert plan is not None
        assert _direct(plan).ecosystem == "aqara"
        assert _direct(plan).protocol == "zigbee"
        assert _final(plan).ecosystem == "google"
        assert _final(plan).protocol == "matter-over-thread"

    def test_matter_thread_bridge_no_target_hub_returns_none(self):
        device = make_device(
            direct_compat=(DirectCompat("aqara", "zigbee"),),
            bridge_compat=(BridgeCompat("aqara", "google", "matter-over-thread"),),
        )
        # aqara hub present but no google matter hub
        plan = find_connection(
            device, True, MAIN, ALL_ECOSYSTEMS,
            frozenset({AQARA_ZIGBEE_MATTER_THREAD_HUB}),
        )
        assert plan is None

    def test_matter_thread_bridge_no_source_hub_returns_none(self):
        device = make_device(
            direct_compat=(DirectCompat("aqara", "zigbee"),),
            bridge_compat=(BridgeCompat("aqara", "google", "matter-over-thread"),),
        )
        # google matter hub present but no aqara hub
        plan = find_connection(
            device, True, MAIN, ALL_ECOSYSTEMS,
            frozenset({GOOGLE_MATTER_THREAD_HUB}),
        )
        assert plan is None

    def test_matter_thread_bridge_source_hub_wrong_protocol(self):
        # aqara hub only supports zigbee, not matter-over-thread - can't bridge
        device = make_device(
            direct_compat=(DirectCompat("aqara", "zigbee"),),
            bridge_compat=(BridgeCompat("aqara", "google", "matter-over-thread"),),
        )
        plan = find_connection(
            device, True, MAIN, ALL_ECOSYSTEMS,
            frozenset({AQARA_ZIGBEE_HUB, GOOGLE_MATTER_THREAD_HUB}),
        )
        # aqara hub supports zigbee so direct to aqara works,
        # but aqara hub doesnt support matter-over-thread so cant bridge
        assert plan is None

    def test_matter_wifi_bridge_source_wifi_direct(self):
        # device connects to aqara via wifi directly, bridges to google via matter-over-wifi
        aqara_matter_wifi_hub = HubType(
            ecosystem="aqara", protocols=frozenset({"matter-over-wifi"})
        )
        google_matter_wifi_hub = HubType(
            ecosystem="google", protocols=frozenset({"matter-over-wifi"})
        )
        device = make_device(
            direct_compat=(DirectCompat("aqara", "wifi"),),
            bridge_compat=(BridgeCompat("aqara", "google", "matter-over-wifi"),),
        )
        plan = find_connection(
            device, True, MAIN, ALL_ECOSYSTEMS,
            frozenset({aqara_matter_wifi_hub, google_matter_wifi_hub}),
        )
        assert plan is not None
        assert _direct(plan).ecosystem == "aqara"
        assert _final(plan).protocol == "matter-over-wifi"


class TestNoMainEcosystemRequired:
    def test_wifi_any_ecosystem_works(self):
        device = make_device(direct_compat=(DirectCompat("aqara", "wifi"),))
        plan = find_connection(device, False, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is not None
        assert _direct(plan).ecosystem == "aqara"
        assert _final(plan) is None

    def test_zigbee_any_ecosystem_hub_available(self):
        device = make_device(direct_compat=(DirectCompat("aqara", "zigbee"),))
        plan = find_connection(device, False, MAIN, ALL_ECOSYSTEMS, frozenset({AQARA_ZIGBEE_HUB}))
        assert plan is not None
        assert _direct(plan).ecosystem == "aqara"
        assert _direct(plan).protocol == "zigbee"

    def test_zigbee_any_ecosystem_no_hub_returns_none(self):
        device = make_device(direct_compat=(DirectCompat("aqara", "zigbee"),))
        plan = find_connection(device, False, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is None

    def test_still_tries_main_ecosystem_first(self):
        device = make_device(
            direct_compat=(
                DirectCompat("aqara", "wifi"),
                DirectCompat("google", "wifi"),
            ),
        )
        plan = find_connection(device, False, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is not None
        # google is main so tried first
        assert _direct(plan).ecosystem == "google"

    def test_ecosystem_not_in_available_returns_none(self):
        device = make_device(direct_compat=(DirectCompat("aqara", "wifi"),))
        plan = find_connection(device, False, MAIN, frozenset({"tuya"}), NO_HUBS)
        assert plan is None

    def test_no_compat_records_returns_none(self):
        device = make_device()
        plan = find_connection(device, False, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is None

    def test_bridge_not_used_when_main_not_required(self):
        # even if bridge exists, we only try direct when main not required
        device = make_device(
            direct_compat=(DirectCompat("aqara", "zigbee"),),
            bridge_compat=(BridgeCompat("aqara", "google", "cloud"),),
        )
        plan = find_connection(device, False, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        # zigbee needs hub, no hub available, so None even though bridge exists
        assert plan is None


class TestMultipleBridgeOptions:
    def test_falls_back_to_second_bridge_if_first_fails(self):
        # two bridge options, first one has no valid source direct compat
        device = make_device(
            direct_compat=(
                DirectCompat("tuya", "wifi"),
            ),
            bridge_compat=(
                BridgeCompat("aqara", "google", "cloud"),  # aqara not available for device
                BridgeCompat("tuya", "google", "cloud"),   # tuya works
            ),
        )
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is not None
        assert _direct(plan).ecosystem == "tuya"
        assert _final(plan).ecosystem == "google"

    def test_no_bridge_works_returns_none(self):
        device = make_device(
            direct_compat=(DirectCompat("aqara", "zigbee"),),
            bridge_compat=(BridgeCompat("aqara", "google", "cloud"),),
        )
        # no hubs at all, zigbee source fails
        plan = find_connection(device, True, MAIN, ALL_ECOSYSTEMS, NO_HUBS)
        assert plan is None
