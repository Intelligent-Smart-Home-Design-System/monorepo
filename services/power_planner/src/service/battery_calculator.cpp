#include <cmath>
#include <string>

#include "power_planner/service/battery_calculator.hpp"

namespace {
    int months_between(const std::string &from, const std::string &to) {
        if (from.size() < 7 || to.size() < 7) {
            return 0;
        }
        const int from_year = std::stoi(from.substr(0, 4));
        const int from_month = std::stoi(from.substr(5, 2));
        const int to_year = std::stoi(to.substr(0, 4));
        const int to_month = std::stoi(to.substr(5, 2));
        return (to_year - from_year) * 12 + (to_month - from_month);
    }
}

namespace power_planner::service {
    BatteryReport BatteryCalculator::calculate(const std::vector<power_planner::model::Device> &devices,
                                               int period_months,
                                               const std::string &current_date) const {
        BatteryReport report;
        report.period_months = period_months;
        for (const auto &device: devices) {
            if (device.power_type != "battery") {
                continue;
            }
            BatteryDeviceReport battery_report;
            if (device.battery_life_months > 0) {
                const int used_months = months_between(device.battery_installed_at, current_date);
                const int months_since_last_replacement = used_months % static_cast<int>(device.battery_life_months);
                battery_report.next_replacement_in_months =
                        static_cast<int>(device.battery_life_months) - months_since_last_replacement;
                battery_report.replacements_in_period =
                        (period_months + months_since_last_replacement) / static_cast<int>(device.battery_life_months);
                battery_report.maintenance_cost = battery_report.replacements_in_period * device.battery_price;
            }
            battery_report.device_id = device.id;
            battery_report.device_type = device.type;
            battery_report.battery_type = device.battery_type;
            battery_report.battery_life_months = device.battery_life_months;
            report.total_maintenance_cost += battery_report.maintenance_cost;
            report.device_reports.push_back(battery_report);
        }
        return report;
    }
}
