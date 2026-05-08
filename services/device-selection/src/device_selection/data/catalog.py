from __future__ import annotations
from dataclasses import dataclass
from typing import Optional, Protocol, Sequence

from device_selection.core.model import Device, DeviceId, DeviceTypeId

class Catalog(Protocol):
    def devices_for_type(self, type_id: DeviceTypeId) -> Sequence[Device]: ...
    def get_device(self, id: DeviceId) -> Optional[Device]: ...

@dataclass(slots=True)
class InMemoryCatalog(Catalog):
    devices_by_id: dict[DeviceId, Device]
    devices_by_type: dict[DeviceTypeId, list[Device]]

    def __init__(self, devices_by_type: dict[DeviceTypeId, list[Device]]) -> InMemoryCatalog:
        self.devices_by_type = devices_by_type
        self.devices_by_id = {}
        for _, devices in devices_by_type.items():
            for d in devices:
                self.devices_by_id[d.device_id] = d

    def devices_for_type(self, type_id: DeviceTypeId) -> Sequence[Device]:
        return self.devices_by_type.get(type_id, [])
    
    def get_device(self, id: DeviceId) -> Optional[Device]:
        if id not in self.devices_by_id:
            return None
        return self.devices_by_id[id]
