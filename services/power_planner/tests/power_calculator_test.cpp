#include <cassert>
#include <cmath>
#include <vector>

#include "power_planner/service/power_calculator.hpp"
#include "power_planner/model/connection.hpp"
#include "power_planner/model/device.hpp"
#include "power_planner/model/power_source.hpp"

namespace {
    bool nearly_equal(double lhs, double rhs, double eps = 1e-6) {
        return std::abs(lhs - rhs) < eps;
    }

    void test_normal_case() {
        using power_planner::service::PowerCalculator;
        using power_planner::model::Connection;
        using power_planner::model::Device;
        using power_planner::model::PowerSource;

        std::vector<Device> devices{
            {"cam_1", "camera", "mains", "", 0.0, "", 0.0, 5.0, 7.0, 220.0},
            {"hub_1", "hub", "mains", "", 0.0, "", 0.0, 10.0, 12.0, 220.0}
        };

        std::vector<PowerSource> power_sources{
            {"socket_1", "socket", 1.0, 220.0}
        };

        std::vector<Connection> connections{
            {"cam_1", "socket_1"},
            {"hub_1", "socket_1"}
        };

        PowerCalculator calculator;
        const auto report = calculator.calculate(devices, power_sources, connections);

        assert(nearly_equal(report.total_nominal_power_w, 15.0));
        assert(nearly_equal(report.total_peak_power_w, 19.0));
        assert(report.source_loads.size() == 1);
        assert(report.unassigned_device_ids.empty());

        const auto &source_report = report.source_loads[0];
        assert(nearly_equal(source_report.nominal_current_a, 15.0 / 220.0));
        assert(nearly_equal(source_report.peak_current_a, 19.0 / 220.0));
        assert(!source_report.overloaded_nominal);
        assert(!source_report.overloaded_peak);
    }

    void test_overload_case() {
        using power_planner::service::PowerCalculator;
        using power_planner::model::Connection;
        using power_planner::model::Device;
        using power_planner::model::PowerSource;

        std::vector<Device> devices{
            {"cam_1", "camera", "mains", "", 0.0, "", 0.0, 20.0, 25.0, 220.0},
            {"cam_2", "camera", "mains", "", 0.0, "", 0.0, 20.0, 25.0, 220.0},
            {"hub_1", "hub", "mains", "", 0.0, "", 0.0, 30.0, 35.0, 220.0}
        };

        std::vector<PowerSource> power_sources{
            {"socket_1", "socket", 0.2, 220.0}
        };

        std::vector<Connection> connections{
            {"cam_1", "socket_1"},
            {"cam_2", "socket_1"},
            {"hub_1", "socket_1"}
        };

        PowerCalculator calculator;
        const auto report = calculator.calculate(devices, power_sources, connections);

        assert(report.source_loads.size() == 1);

        const auto &source_report = report.source_loads[0];
        assert(source_report.overloaded_nominal);
        assert(source_report.overloaded_peak);
    }

    void test_unassigned_device_case() {
        using power_planner::service::PowerCalculator;
        using power_planner::model::Connection;
        using power_planner::model::Device;
        using power_planner::model::PowerSource;

        std::vector<Device> devices{
            {"cam_1", "camera", "mains", "", 0.0, "", 0.0, 5.0, 7.0, 220.0},
            {"hub_1", "hub", "mains", "", 0.0, "", 0.0, 10.0, 12.0, 220.0}
        };

        std::vector<PowerSource> power_sources{
            {"socket_1", "socket", 1.0, 220.0}
        };

        std::vector<Connection> connections{
            {"cam_1", "socket_1"}
        };

        PowerCalculator calculator;
        const auto report = calculator.calculate(devices, power_sources, connections);

        assert(report.unassigned_device_ids.size() == 1);
        assert(report.unassigned_device_ids[0] == "hub_1");
    }
}

int main() {
    test_normal_case();
    test_overload_case();
    test_unassigned_device_case();
    return 0;
}
