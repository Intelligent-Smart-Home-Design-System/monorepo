#pragma once

#include <vector>

#include "power_planner/model/device.hpp"
#include "power_planner/model/power_source.hpp"
#include "power_planner/model/connection.hpp"

namespace power_planner::model {
    struct PlannerInput {
        std::vector<Device> devices;
        std::vector<PowerSource> power_sources;
        std::vector<Connection> connections;
    };
}
