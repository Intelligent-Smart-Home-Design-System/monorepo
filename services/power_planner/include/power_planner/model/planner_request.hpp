#pragma once

#include "power_planner/model/planner_input.hpp"
#include "power_planner/model/run_options.hpp"

namespace power_planner::model {
    struct PlannerRequest {
        PlannerInput input;
        RunOptions options;
    };
}
