#pragma once

#include <vector>
#include <string>

#include "battery_report.hpp"
#include "power_planner/model/device.hpp"

namespace power_planner::service {
    class BatteryCalculator {
    public:
        BatteryReport calculate(const std::vector<power_planner::model::Device> &devices,
                                int period_months,
                                const std::string &current_date) const;
    };
}
