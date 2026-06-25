#include <cassert>

#include "power_planner/service/request_loader.hpp"

namespace {
    void test_sample_input() {
        power_planner::service::RequestLoader loader;
        const auto request = loader.load("data/sample_input.json");
        assert(request.input.devices.size() == 4);
        assert(request.input.power_sources.size() == 1);
        assert(request.input.connections.size() == 3);
        assert(request.input.devices[0].id == "cam_1");
        assert(request.input.devices[0].type == "camera");
        assert(request.input.devices[0].power_type == "mains");
        assert(request.input.devices[0].nominal_power_w == 5.0);
        assert(request.input.devices[0].peak_power_w == 7.0);
        assert(request.input.devices[3].id == "motion_sensor_1");
        assert(request.input.devices[3].power_type == "battery");
        assert(request.input.devices[3].battery_type == "AA");
        assert(request.input.devices[3].battery_life_months == 12.0);
        assert(request.input.devices[3].battery_price == 50.0);
        assert(request.input.power_sources[0].id == "socket_1");
        assert(request.input.power_sources[0].kind == "socket");
        assert(request.input.power_sources[0].max_current_a == 0.08);
        assert(request.input.connections[0].device_id == "cam_1");
        assert(request.input.connections[0].power_source_id == "socket_1");
        assert(request.options.battery_period_months == 12);
        assert(request.options.current_date == "2026-06-01");
    }
}

int main() {
    test_sample_input();
    return 0;
}
