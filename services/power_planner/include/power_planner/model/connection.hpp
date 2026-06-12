#pragma once

#include <string>

namespace power_planner::model {
    struct Connection {
        std::string device_id;
        std::string power_source_id;
    };
}
