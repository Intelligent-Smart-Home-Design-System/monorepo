#pragma once

#include <string>
#include <vector>

#include "power_planner/model/connection.hpp"
#include "power_planner/model/device.hpp"
#include "power_planner/model/power_source.hpp"

namespace power_planner::service {
    struct PowerAssignment {
        std::vector<power_planner::model::Connection> connections;
        std::vector<std::string> unassigned_device_ids;
    };

    class PowerAssignmentPlanner {
    public:
        PowerAssignment plan(const std::vector<power_planner::model::Device> &devices,
                             const std::vector<power_planner::model::PowerSource> &power_sources) const;
    };
}
