#pragma once

#include <vector>

#include "power_planner/service/power_report.hpp"
#include "power_planner/model/connection.hpp"
#include "power_planner/model/device.hpp"
#include "power_planner/model/power_source.hpp"

namespace power_planner::service {
    class PowerCalculator {
    public:
        PowerReport calculate(const std::vector<power_planner::model::Device> &devices,
                                    const std::vector<power_planner::model::PowerSource> &power_sources,
                                    const std::vector<power_planner::model::Connection> &connections) const;
    };
}
