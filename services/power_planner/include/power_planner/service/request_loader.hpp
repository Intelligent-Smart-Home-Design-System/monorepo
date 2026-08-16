#pragma once

#include <string>

#include "power_planner/model/planner_request.hpp"

namespace power_planner::service {
    class RequestLoader {
    public:
        power_planner::model::PlannerRequest load(const std::string &file_path) const;
    };
}
