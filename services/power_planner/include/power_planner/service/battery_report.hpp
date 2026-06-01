#pragma once

#include <string>
#include <vector>

namespace power_planner::service {
    struct BatteryDeviceReport {
        std::string device_id;
        std::string device_type;
        std::string battery_type;
        double battery_life_months;
        double next_replacement_in_months;
    };

    struct BatteryReport {
        std::vector<BatteryDeviceReport> device_reports;
    };
}
