#include <cassert>
#include <vector>

#include "power_planner/model/device.hpp"
#include "power_planner/model/power_source.hpp"
#include "power_planner/service/power_assignment.hpp"

namespace {
    void test_assigns_mains_devices_to_available_sources() {
        using power_planner::model::Device;
        using power_planner::model::PowerSource;
        using power_planner::service::PowerAssignmentPlanner;
        std::vector<Device> devices{
            {"cam_1", "camera", "mains", "", 0.0, "", 0.0, 5.0, 10.0, 220.0},
            {"hub_1", "hub", "mains", "", 0.0, "", 0.0, 10.0, 20.0, 220.0},
            {"motion_sensor_1", "motion_sensor", "battery", "AA", 12.0, "", 50.0, 0.0, 0.0, 1.5}
        };
        std::vector<PowerSource> sources{
            {"socket_1", "socket", 1.0, 220.0}
        };
        PowerAssignmentPlanner planner;
        const auto assignment = planner.plan(devices, sources);
        assert(assignment.connections.size() == 2);
        assert(assignment.unassigned_device_ids.empty());
        assert(assignment.connections[0].device_id == "hub_1");
        assert(assignment.connections[0].power_source_id == "socket_1");
        assert(assignment.connections[1].device_id == "cam_1");
        assert(assignment.connections[1].power_source_id == "socket_1");
    }

    void test_marks_device_as_unassigned_when_source_is_too_weak() {
        using power_planner::model::Device;
        using power_planner::model::PowerSource;
        using power_planner::service::PowerAssignmentPlanner;
        std::vector<Device> devices{
            {"big_camera", "camera", "mains", "", 0.0, "", 0.0, 20.0, 50.0, 220.0}
        };
        std::vector<PowerSource> sources{
            {"socket", "socket", 0.1, 220.0}
        };
        PowerAssignmentPlanner planner;
        const auto assignment = planner.plan(devices, sources);
        assert(assignment.connections.empty());
        assert(assignment.unassigned_device_ids.size() == 1);
        assert(assignment.unassigned_device_ids[0] == "big_camera");
    }
}

int main() {
    test_assigns_mains_devices_to_available_sources();
    test_marks_device_as_unassigned_when_source_is_too_weak();
    return 0;
}
