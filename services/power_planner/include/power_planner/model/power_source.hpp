#pragma once

#include <string>

namespace power_planner::model {
    struct PowerSource {
        std::string id;
        std::string kind;
        double max_current_a{};
        double voltage_v{220};
    };
}
