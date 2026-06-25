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
        int replacements_in_period{};
        double maintenance_cost{};
    };

    struct BatteryReport {
        std::vector<BatteryDeviceReport> device_reports;
        int period_months{};
        double total_maintenance_cost{};
    };
}
