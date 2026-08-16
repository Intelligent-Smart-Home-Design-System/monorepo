#pragma once

#include <string>
#include <vector>

namespace power_planner::service {
    struct PowerSourceReport {
        std::string power_source_id;
        double nominal_power_w {};
        double peak_power_w {};
        double nominal_current_a {};
        double peak_current_a {};
        bool overloaded_nominal {false};
        bool overloaded_peak {false};
    };
    struct PowerReport {
        double total_nominal_power_w {};
        double total_peak_power_w {};
        std::vector<PowerSourceReport> source_loads {};
        std::vector<std::string> unassigned_device_ids;
    };
}