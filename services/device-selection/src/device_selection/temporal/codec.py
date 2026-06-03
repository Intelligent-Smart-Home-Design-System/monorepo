from __future__ import annotations

from google.protobuf.struct_pb2 import Value

from device_selection.core.model import (
    ConnectionInfo,
    DeviceRequirement,
    DeviceSelectionRequest,
    Filter,
    FilterOp,
    ParetoPoint,
    SolutionItem,
)
from device_selection.proto import iot_opt_pb2 as pb

_OP_FROM_PROTO: dict[int, FilterOp] = {
    pb.OP_EQ:       FilterOp.EQ,
    pb.OP_NEQ:      FilterOp.NEQ,
    pb.OP_GT:       FilterOp.GT,
    pb.OP_GTE:      FilterOp.GTE,
    pb.OP_LT:       FilterOp.LT,
    pb.OP_LTE:      FilterOp.LTE,
    pb.OP_CONTAINS: FilterOp.CONTAINS,
    pb.OP_EXISTS:   FilterOp.EXISTS,
}


def _value_to_py(v: Value) -> str | int | float | None:
    kind = v.WhichOneof("kind")
    if kind == "string_value":
        return v.string_value
    if kind == "number_value":
        n = v.number_value
        return int(n) if n == int(n) else n
    if kind == "bool_value":
        return v.bool_value
    return None


def _py_to_value(v: str | int | float | None) -> Value:
    out = Value()
    if isinstance(v, str):
        out.string_value = v
    elif isinstance(v, bool):
        out.bool_value = v
    elif isinstance(v, int):
        out.number_value = float(v)
    elif isinstance(v, float):
        out.number_value = v
    else:
        out.null_value = 0
    return out


def request_from_proto(msg: pb.DeviceSelectionRequest) -> DeviceSelectionRequest:
    requirements = tuple(
        DeviceRequirement(
            requirement_id=r.requirement_id,
            device_type=r.device_type,
            count=r.count,
            connect_to_main_ecosystem=r.connect_to_main_ecosystem,
            filters=tuple(
                Filter(
                    field=f.field,
                    op=_OP_FROM_PROTO.get(f.op, FilterOp.EQ),
                    value=_value_to_py(f.value),
                )
                for f in r.filters
            ),
        )
        for r in msg.device_requirements
    )
    return DeviceSelectionRequest(
        main_ecosystem=msg.main_ecosystem,
        budget=msg.budget,
        requirements=requirements,
        include_ecosystems=frozenset(msg.include_ecosystems),
        exclude_ecosystems=frozenset(msg.exclude_ecosystems),
        max_solutions=msg.max_solutions if msg.HasField("max_solutions") else 7,
        random_seed=msg.random_seed if msg.HasField("random_seed") else None,
        time_budget_seconds=(
            msg.time_budget_seconds
            if msg.HasField("time_budget_seconds")
            else 180.0
        ),
    )


def _conn_info_to_proto(info: ConnectionInfo) -> pb.ConnectionInfo:
    msg = pb.ConnectionInfo(
        ecosystem=info.ecosystem,
        protocol=info.protocol,
    )
    if info.hub_solution_item_id is not None:
        msg.hub_selected_listing_id = info.hub_solution_item_id
    return msg


def _solution_item_to_proto(item: SolutionItem) -> pb.SelectedListing:
    msg = pb.SelectedListing(
        selected_listing_id=item.id,
        device_id=item.device.device_id,
        device_quality=item.device.quality,
        connection_direct=_conn_info_to_proto(item.connection.connection_direct),
        extracted_listing_id=item.device.source_listing_id,
        devices_per_listing=1,
        units_to_buy=item.quantity,
        unit_price=item.device.price,
    )
    if item.requirement_id is not None:
        msg.requirement_id = item.requirement_id
    if item.connection.connection_final is not None:
        msg.connection_final.CopyFrom(
            _conn_info_to_proto(item.connection.connection_final)
        )
    return msg


def pareto_point_to_proto(point: ParetoPoint) -> pb.ParetoPoint:
    return pb.ParetoPoint(
        listings=[_solution_item_to_proto(item) for item in point.items],
        total_cost=point.total_cost,
        avg_quality=point.avg_quality,
        num_ecosystems=point.num_ecosystems,
        num_hubs=point.num_hubs,
        is_recommended=False, # not implemented
    )


def response_to_proto(points: list[ParetoPoint]) -> pb.DeviceSelectionResponse:
    return pb.DeviceSelectionResponse(
        pareto_front=[
            pareto_point_to_proto(p)
            for i, p in enumerate(points)
        ]
    )
