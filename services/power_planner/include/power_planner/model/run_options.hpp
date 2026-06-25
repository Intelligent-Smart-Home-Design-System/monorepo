#pragma once

#include <string>

namespace power_planner::model {
    struct RunOptions {
        int battery_period_months{};
        std::string current_date;
    };
}
