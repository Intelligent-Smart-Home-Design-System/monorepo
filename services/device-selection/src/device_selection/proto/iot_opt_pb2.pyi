from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Op(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OP_UNSPECIFIED: _ClassVar[Op]
    OP_EQ: _ClassVar[Op]
    OP_NEQ: _ClassVar[Op]
    OP_GT: _ClassVar[Op]
    OP_GTE: _ClassVar[Op]
    OP_LT: _ClassVar[Op]
    OP_LTE: _ClassVar[Op]
    OP_CONTAINS: _ClassVar[Op]
    OP_EXISTS: _ClassVar[Op]
OP_UNSPECIFIED: Op
OP_EQ: Op
OP_NEQ: Op
OP_GT: Op
OP_GTE: Op
OP_LT: Op
OP_LTE: Op
OP_CONTAINS: Op
OP_EXISTS: Op

class Filter(_message.Message):
    __slots__ = ("field", "op", "value")
    FIELD_FIELD_NUMBER: _ClassVar[int]
    OP_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    field: str
    op: Op
    value: _struct_pb2.Value
    def __init__(self, field: _Optional[str] = ..., op: _Optional[_Union[Op, str]] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class DeviceRequirement(_message.Message):
    __slots__ = ("requirement_id", "device_type", "count", "connect_to_main_ecosystem", "filters")
    REQUIREMENT_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_TYPE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    CONNECT_TO_MAIN_ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    FILTERS_FIELD_NUMBER: _ClassVar[int]
    requirement_id: int
    device_type: str
    count: int
    connect_to_main_ecosystem: bool
    filters: _containers.RepeatedCompositeFieldContainer[Filter]
    def __init__(self, requirement_id: _Optional[int] = ..., device_type: _Optional[str] = ..., count: _Optional[int] = ..., connect_to_main_ecosystem: bool = ..., filters: _Optional[_Iterable[_Union[Filter, _Mapping]]] = ...) -> None: ...

class DeviceSelectionRequest(_message.Message):
    __slots__ = ("main_ecosystem", "budget", "device_requirements", "include_ecosystems", "exclude_ecosystems", "max_solutions", "random_seed", "time_budget_seconds")
    MAIN_ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    DEVICE_REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ECOSYSTEMS_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_ECOSYSTEMS_FIELD_NUMBER: _ClassVar[int]
    MAX_SOLUTIONS_FIELD_NUMBER: _ClassVar[int]
    RANDOM_SEED_FIELD_NUMBER: _ClassVar[int]
    TIME_BUDGET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    main_ecosystem: str
    budget: float
    device_requirements: _containers.RepeatedCompositeFieldContainer[DeviceRequirement]
    include_ecosystems: _containers.RepeatedScalarFieldContainer[str]
    exclude_ecosystems: _containers.RepeatedScalarFieldContainer[str]
    max_solutions: int
    random_seed: int
    time_budget_seconds: float
    def __init__(self, main_ecosystem: _Optional[str] = ..., budget: _Optional[float] = ..., device_requirements: _Optional[_Iterable[_Union[DeviceRequirement, _Mapping]]] = ..., include_ecosystems: _Optional[_Iterable[str]] = ..., exclude_ecosystems: _Optional[_Iterable[str]] = ..., max_solutions: _Optional[int] = ..., random_seed: _Optional[int] = ..., time_budget_seconds: _Optional[float] = ...) -> None: ...

class ConnectionInfo(_message.Message):
    __slots__ = ("ecosystem", "protocol", "hub_selected_listing_id")
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    HUB_SELECTED_LISTING_ID_FIELD_NUMBER: _ClassVar[int]
    ecosystem: str
    protocol: str
    hub_selected_listing_id: int
    def __init__(self, ecosystem: _Optional[str] = ..., protocol: _Optional[str] = ..., hub_selected_listing_id: _Optional[int] = ...) -> None: ...

class SelectedListing(_message.Message):
    __slots__ = ("selected_listing_id", "requirement_id", "device_id", "device_quality", "connection_direct", "connection_final", "extracted_listing_id", "devices_per_listing", "units_to_buy", "unit_price")
    SELECTED_LISTING_ID_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_QUALITY_FIELD_NUMBER: _ClassVar[int]
    CONNECTION_DIRECT_FIELD_NUMBER: _ClassVar[int]
    CONNECTION_FINAL_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_LISTING_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICES_PER_LISTING_FIELD_NUMBER: _ClassVar[int]
    UNITS_TO_BUY_FIELD_NUMBER: _ClassVar[int]
    UNIT_PRICE_FIELD_NUMBER: _ClassVar[int]
    selected_listing_id: int
    requirement_id: int
    device_id: int
    device_quality: float
    connection_direct: ConnectionInfo
    connection_final: ConnectionInfo
    extracted_listing_id: int
    devices_per_listing: int
    units_to_buy: int
    unit_price: float
    def __init__(self, selected_listing_id: _Optional[int] = ..., requirement_id: _Optional[int] = ..., device_id: _Optional[int] = ..., device_quality: _Optional[float] = ..., connection_direct: _Optional[_Union[ConnectionInfo, _Mapping]] = ..., connection_final: _Optional[_Union[ConnectionInfo, _Mapping]] = ..., extracted_listing_id: _Optional[int] = ..., devices_per_listing: _Optional[int] = ..., units_to_buy: _Optional[int] = ..., unit_price: _Optional[float] = ...) -> None: ...

class ParetoPoint(_message.Message):
    __slots__ = ("listings", "total_cost", "avg_quality", "num_ecosystems", "num_hubs", "is_recommended")
    LISTINGS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COST_FIELD_NUMBER: _ClassVar[int]
    AVG_QUALITY_FIELD_NUMBER: _ClassVar[int]
    NUM_ECOSYSTEMS_FIELD_NUMBER: _ClassVar[int]
    NUM_HUBS_FIELD_NUMBER: _ClassVar[int]
    IS_RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    listings: _containers.RepeatedCompositeFieldContainer[SelectedListing]
    total_cost: float
    avg_quality: float
    num_ecosystems: int
    num_hubs: int
    is_recommended: bool
    def __init__(self, listings: _Optional[_Iterable[_Union[SelectedListing, _Mapping]]] = ..., total_cost: _Optional[float] = ..., avg_quality: _Optional[float] = ..., num_ecosystems: _Optional[int] = ..., num_hubs: _Optional[int] = ..., is_recommended: bool = ...) -> None: ...

class DeviceSelectionResponse(_message.Message):
    __slots__ = ("pareto_front",)
    PARETO_FRONT_FIELD_NUMBER: _ClassVar[int]
    pareto_front: _containers.RepeatedCompositeFieldContainer[ParetoPoint]
    def __init__(self, pareto_front: _Optional[_Iterable[_Union[ParetoPoint, _Mapping]]] = ...) -> None: ...
