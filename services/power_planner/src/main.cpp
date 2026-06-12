#include <iostream>
#include <vector>

#include "power_planner/service/power_assignment.hpp"
#include "power_planner/service/power_calculator.hpp"
#include "power_planner/model/connection.hpp"
#include "power_planner/model/device.hpp"
#include "power_planner/model/power_source.hpp"
#include "power_planner/service/battery_calculator.hpp"
#include "power_planner/service/request_loader.hpp"
#include "power_planner/service/result_writer.hpp"

int main() {
    using power_planner::service::PowerCalculator;
    using power_planner::model::Connection;
    using power_planner::model::Device;
    using power_planner::model::PowerSource;
    using power_planner::service::BatteryCalculator;
    using power_planner::service::RequestLoader;
    using power_planner::service::PowerAssignmentPlanner;
    using power_planner::service::ResultWriter;


    RequestLoader loader;
    auto request = loader.load("data/sample_input.json");
    PowerAssignmentPlanner assignment_planner;
    const auto assignment = assignment_planner.plan(request.input.devices, request.input.power_sources);
    std::cout << "Power assignment:\n";
    for (const auto &connection: assignment.connections) {
        std::cout << connection.device_id << " -> " << connection.power_source_id << "\n";
    }
    PowerCalculator calculator;
    const auto report = calculator.calculate(request.input.devices, request.input.power_sources,
                                             assignment.connections);

    std::cout << "Total nominal power: " << report.total_nominal_power_w << "W\n";
    std::cout << "Total peak power: " << report.total_peak_power_w << "W\n";

    for (const auto &node_load: report.source_loads) {
        std::cout << "Node: " << node_load.power_source_id << "\n";
        std::cout << "Nominal current: " << node_load.nominal_current_a << " A\n";
        std::cout << "Peak current: " << node_load.peak_current_a << " A\n";
        std::cout << "Overloaded nomial: " << (node_load.overloaded_nominal ? "yes" : "no") << "\n";
        std::cout << " Overloaded peak: " << (node_load.overloaded_peak ? "yes" : "no") << "\n";
    }

    if (!report.unassigned_device_ids.empty()) {
        std::cout << "Unassigned devices:\n";
        for (const auto &id: report.unassigned_device_ids) {
            std::cout << " " << id << "\n";
        }
    }
    BatteryCalculator battery_calculator;
    const auto battery_report = battery_calculator.calculate(request.input.devices,
                                                             request.options.battery_period_months,
                                                             request.options.current_date);
    std::cout << "Battery report: " << "\n";
    for (const auto &device_report: battery_report.device_reports) {
        std::cout << "Device: " << device_report.device_id << "\n";
        std::cout << "Battery type: " << device_report.battery_type << "\n";
        std::cout << "Next replacement in: " << device_report.next_replacement_in_months << " months\n";
        std::cout << "Replacements in period: " << device_report.replacements_in_period << " times\n";
        std::cout << "Maintenance cost: " << device_report.maintenance_cost << "\n";
        std::cout << "\n";
    }
    std::cout << "Total cost: " << battery_report.total_maintenance_cost << "\n";
    ResultWriter writer;
    std::cout << "JSON result:\n";
    std::cout << writer.write(assignment, report, battery_report) << "\n";
    return 0;
}
