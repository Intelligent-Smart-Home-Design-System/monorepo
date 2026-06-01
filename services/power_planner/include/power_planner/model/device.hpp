#pragma once

#include <string>

namespace power_planner::model {
    struct Device {
        std::string id;
        std::string type;
        std::string power_type;
        std::string battery_type;
        double battery_life_months{};
        double nominal_power_w{};
        double peak_power_w{};
        double voltage_v{220};
    };
}
