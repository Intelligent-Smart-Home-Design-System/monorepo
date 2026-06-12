#pragma once

#include <string>

#include "power_planner/service/battery_report.hpp"
#include "power_planner/service/power_assignment.hpp"
#include "power_planner/service/power_report.hpp"

namespace power_planner::service {
    class ResultWriter {
    public:
        std::string write(const PowerAssignment &assignment,
                          const PowerReport &power_report,
                          const BatteryReport &battery_report) const;
    };
}
