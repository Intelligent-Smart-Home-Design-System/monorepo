#include "power_planner/service/result_writer.hpp"

#include <nlohmann/json.hpp>

namespace power_planner::service {
    std::string ResultWriter::write(const PowerAssignment &assignment,
                                    const PowerReport &power_report,
                                    const BatteryReport &battery_report) const {
        nlohmann::json json;

        json["power_assignment"]["connections"] = nlohmann::json::array();
        for (const auto &connection: assignment.connections) {
            json["power_assignment"]["connections"].push_back({
                {"device_id", connection.device_id},
                {"power_source_id", connection.power_source_id}
            });
        }

        json["power_assignment"]["unassigned_device_ids"] = assignment.unassigned_device_ids;

        json["power_report"]["total_nominal_power_w"] = power_report.total_nominal_power_w;
        json["power_report"]["total_peak_power_w"] = power_report.total_peak_power_w;
        json["power_report"]["unassigned_device_ids"] = power_report.unassigned_device_ids;

        json["power_report"]["source_loads"] = nlohmann::json::array();
        for (const auto &source_load: power_report.source_loads) {
            json["power_report"]["source_loads"].push_back({
                {"power_source_id", source_load.power_source_id},
                {"nominal_power_w", source_load.nominal_power_w},
                {"peak_power_w", source_load.peak_power_w},
                {"nominal_current_a", source_load.nominal_current_a},
                {"peak_current_a", source_load.peak_current_a},
                {"overloaded_nominal", source_load.overloaded_nominal},
                {"overloaded_peak", source_load.overloaded_peak}
            });
        }

        json["battery_report"]["period_months"] = battery_report.period_months;
        json["battery_report"]["total_maintenance_cost"] = battery_report.total_maintenance_cost;
        json["battery_report"]["devices"] = nlohmann::json::array();

        for (const auto &device_report: battery_report.device_reports) {
            json["battery_report"]["devices"].push_back({
                {"device_id", device_report.device_id},
                {"device_type", device_report.device_type},
                {"battery_type", device_report.battery_type},
                {"battery_life_months", device_report.battery_life_months},
                {"next_replacement_in_months", device_report.next_replacement_in_months},
                {"replacements_in_period", device_report.replacements_in_period},
                {"maintenance_cost", device_report.maintenance_cost}
            });
        }

        return json.dump(2);
    }
}
