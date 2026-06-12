#include <cassert>
#include <string>

#include <nlohmann/json.hpp>

#include "power_planner/service/result_writer.hpp"

namespace {
    void test_writes_result_as_json() {
        power_planner::service::PowerAssignment assignment;
        assignment.connections.push_back({"cam_1", "socket_1"});
        assignment.unassigned_device_ids.push_back("hub_1");

        power_planner::service::PowerReport power_report;
        power_report.total_nominal_power_w = 5.0;
        power_report.total_peak_power_w = 7.0;
        power_report.unassigned_device_ids.push_back("hub_1");
        power_report.source_loads.push_back({
            "socket_1",
            5.0,
            7.0,
            0.02,
            0.03,
            false,
            false
        });

        power_planner::service::BatteryReport battery_report;
        battery_report.period_months = 12;
        battery_report.total_maintenance_cost = 50.0;
        battery_report.device_reports.push_back({
            "motion_sensor_1",
            "motion_sensor",
            "AA",
            12.0,
            7.0,
            1,
            50.0
        });

        power_planner::service::ResultWriter writer;
        const auto output = writer.write(assignment, power_report, battery_report);
        const auto json = nlohmann::json::parse(output);

        assert(json["power_assignment"]["connections"][0]["device_id"] == "cam_1");
        assert(json["power_assignment"]["unassigned_device_ids"][0] == "hub_1");

        assert(json["power_report"]["total_nominal_power_w"] == 5.0);
        assert(json["power_report"]["source_loads"][0]["power_source_id"] == "socket_1");
        assert(json["power_report"]["source_loads"][0]["overloaded_peak"] == false);

        assert(json["battery_report"]["period_months"] == 12);
        assert(json["battery_report"]["devices"][0]["device_id"] == "motion_sensor_1");
        assert(json["battery_report"]["devices"][0]["next_replacement_in_months"] == 7.0);
    }
}

int main() {
    test_writes_result_as_json();
    return 0;
}
