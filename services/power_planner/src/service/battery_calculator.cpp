#pragma once

#include "power_planner/service/battery_calculator.hpp"

namespace power_planner::service {
    BatteryReport BatteryCalculator::calculate(const std::vector<power_planner::model::Device> &devices) const {
        BatteryReport report;
        for (const auto &device: devices) {
            if (device.power_type != "battery") {
                continue;
            }
            BatteryDeviceReport battery_report;
            battery_report.device_id = device.id;
            battery_report.device_type = device.type;
            battery_report.battery_type = device.battery_type;
            battery_report.battery_life_months = device.battery_life_months;
            battery_report.next_replacement_in_months = device.battery_life_months;
            report.device_reports.push_back(battery_report);
        }
        return report;
    }
}
