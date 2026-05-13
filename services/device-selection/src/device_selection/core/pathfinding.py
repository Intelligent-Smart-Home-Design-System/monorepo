from __future__ import annotations

from typing import Optional

from device_selection.core.model import (
    BridgeCompat,
    ConnectionInfo,
    ConnectionPlan,
    Device,
    DirectCompat,
    EcosystemId,
    HubType,
    ProtocolId,
)

# protocols that don't require a hub for connection
_HUBLESS_PROTOCOLS = frozenset({"wifi", "cloud"})


def hub_required(protocol: ProtocolId) -> bool:
    return protocol not in _HUBLESS_PROTOCOLS


def _find_hub_type(
    ecosystem: EcosystemId,
    protocol: ProtocolId,
    available_hub_types: frozenset[HubType],
) -> Optional[HubType]:
    """Find a hub type in the given ecosystem that supports the given protocol."""
    for hub_type in available_hub_types:
        if hub_type.ecosystem == ecosystem and protocol in hub_type.protocols:
            return hub_type
    return None


def _try_direct(
    dc: DirectCompat,
    available_ecosystems: frozenset[EcosystemId],
    available_hub_types: frozenset[HubType],
) -> Optional[ConnectionInfo]:
    """
    Try to build a direct ConnectionInfo for a given DirectCompat record.
    Returns None if ecosystem is not available or hub is required but not available.
    """
    if dc.ecosystem not in available_ecosystems:
        return None

    if hub_required(dc.protocol):
        hub = _find_hub_type(dc.ecosystem, dc.protocol, available_hub_types)
        if hub is None:
            return None
        return ConnectionInfo(
            ecosystem=dc.ecosystem,
            protocol=dc.protocol,
            hub_solution_item_id=None,  # resolved later when actual hub device is chosen
        )

    return ConnectionInfo(
        ecosystem=dc.ecosystem,
        protocol=dc.protocol,
        hub_solution_item_id=None,
    )


def find_hub_connection(
    hub: Device,
    target_ecosystem: EcosystemId,
) -> Optional[ConnectionPlan]:
    """
    Find a direct connection for a hub device to its target ecosystem.
    Only considers hubless protocols (wifi) since
    hubs connect to their ecosystem directly, not via another hub.
    """
    for dc in hub.direct_compat:
        if dc.ecosystem != target_ecosystem:
            continue
        if hub_required(dc.protocol):
            continue
        return ConnectionPlan(
            connection_direct=ConnectionInfo(
                ecosystem=dc.ecosystem,
                protocol=dc.protocol,
                hub_solution_item_id=None,
            )
        )
    return None


def find_connection(
    device: Device,
    require_main_ecosystem: bool,
    main_ecosystem: EcosystemId,
    available_ecosystems: frozenset[EcosystemId],
    available_hub_types: frozenset[HubType],
) -> Optional[ConnectionPlan]:
    """
    Try to find a valid connection plan for the device.
    1. Try direct connection to main_ecosystem first. If not found:

    If require_main_ecosystem is True:
      - Then try bridge connections: device -> source_ecosystem -> main_ecosystem
    If require_main_ecosystem is False:
      - Try any direct connection to any available ecosystem
    """
    # 1. try direct to main ecosystem
    for dc in device.direct_compat:
        if dc.ecosystem != main_ecosystem:
            continue
        conn = _try_direct(dc, available_ecosystems, available_hub_types)
        if conn is not None:
            return ConnectionPlan(connection_direct=conn)

    if require_main_ecosystem:
        # 2. try bridge connections
        for bc in device.bridge_compat:
            if bc.target_ecosystem != main_ecosystem:
                continue
            if bc.source_ecosystem not in available_ecosystems:
                continue

            # find a direct compat record for the source ecosystem
            source_conn: Optional[ConnectionInfo] = None
            for dc in device.direct_compat:
                if dc.ecosystem != bc.source_ecosystem:
                    continue
                source_conn = _try_direct(dc, available_ecosystems, available_hub_types)
                if source_conn is not None:
                    break

            if source_conn is None:
                continue

            # for cloud bridge protocol, no extra hub needed
            if bc.protocol == "cloud":
                return ConnectionPlan(
                    connection_direct=source_conn,
                    connection_final=ConnectionInfo(
                        ecosystem=main_ecosystem,
                        protocol=bc.protocol,
                        hub_solution_item_id=None,
                    ),
                )

            # for non-cloud bridge protocol, need a hub in both ecosystems
            # that supports the bridge protocol
            hub_target = _find_hub_type(main_ecosystem, bc.protocol, available_hub_types)
            bridge_hub_source = _find_hub_type(bc.source_ecosystem, bc.protocol, available_hub_types)
            if bridge_hub_source is None or hub_target is None:
                continue

            return ConnectionPlan(
                connection_direct=source_conn,
                connection_final=ConnectionInfo(
                    ecosystem=main_ecosystem,
                    protocol=bc.protocol,
                    hub_solution_item_id=None,
                ),
            )

        return None

    else:
        # 2. just find any working direct connection
        for dc in device.direct_compat:
            conn = _try_direct(dc, available_ecosystems, available_hub_types)
            if conn is not None:
                return ConnectionPlan(connection_direct=conn)
        return None
