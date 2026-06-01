#include <iostream>
#include <vector>

#include "power_planner/service/power_calculator.hpp"
#include "power_planner/model/connection.hpp"
#include "power_planner/model/device.hpp"
#include "power_planner/model/power_source.hpp"
#include "power_planner/service/battery_calculator.hpp"

int main() {
    using power_planner::service::PowerCalculator;
    using power_planner::model::Connection;
    using power_planner::model::Device;
    using power_planner::model::PowerSource;
    using power_planner::service::BatteryCalculator;

    std::vector<Device> devices{
        {"cam_1", "camera", "mains", "", 0.0, 5.0, 7.0, 220.0},
        {"cam_2", "camera", "mains", "", 0.0, 5.0, 7.0, 220.0},
        {"hub_1", "hub", "mains", "", 0.0, 12.0, 18.0, 220.0},
        {"motion_sensor_1", "motion_sensor", "battery", "AA", 12, 1.5, 1.5, 1.5}
    };

    std::vector<PowerSource> power_sources{
        {"socket_1", "soket", 0.08, 220.0}
    };

    std::vector<Connection> connections{
        {"cam_1", "socket_1"},
        {"cam_2", "socket_1"},
        {"hub_1", "socket_1"}
    };
    PowerCalculator calculator;
    const auto report = calculator.calculate(devices, power_sources, connections);

    std::cout << "Total nominal power: " << report.total_nominal_power_w << "W\n";
    std::cout << "Total peak power: " << report.total_peak_power_w << "W\n";

    for (const auto &node_load: report.source_loads) {
        std::cout << "Node: " << node_load.power_source_id << "\n";
        std::cout << " Nominal current: " << node_load.nominal_current_a << " A\n";
        std::cout << " Peak current: " << node_load.peak_current_a << " A\n";
        std::cout << " Overloaded nomial: " << (node_load.overloaded_nominal ? "yes" : "no") << "\n";
        std::cout << " Overloaded peak: " << (node_load.overloaded_peak ? "yes" : "no") << "\n";
    }

    if (!report.unassigned_device_ids.empty()) {
        std::cout << "Unassigned devices:\n";
        for (const auto &id: report.unassigned_device_ids) {
            std::cout << " " << id << "\n";
        }
    }
    BatteryCalculator battery_calculator;
    const auto battery_report = battery_calculator.calculate(devices);
    std::cout << "Battery report: " << "\n";
    for (const auto& device_report : battery_report.device_reports) {
        std::cout << "Device: " << device_report.device_id << "\n";
        std::cout << "Battery type: " << device_report.battery_type << "\n";
        std::cout << "Next replacement in: " << device_report.next_replacement_in_months << " months\n";
        std::cout << "\n";
    }
    return 0;
}
