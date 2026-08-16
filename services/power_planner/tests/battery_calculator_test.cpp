#include <cassert>
#include <vector>

#include "power_planner/service/battery_calculator.hpp"
#include "power_planner/model/device.hpp"

namespace {
    void test_battery_devices_are_reported() {
        using power_planner::model::Device;
        using power_planner::service::BatteryCalculator;

        std::vector<Device> devices{
            {"motion_sensor_1", "motion_sensor", "battery", "AA", 12.0, "", 50.0, 0.0, 0.0, 0.0},
            {"door_sensor_1", "door_sensor", "battery", "CR2032", 24.0, "", 120.0, 0.0, 0.0, 0.0}
        };

        BatteryCalculator calculator;

        const auto report = calculator.calculate(devices, 24, "2026-06-01");

        assert(report.device_reports.size() == 2);
        assert(report.period_months == 24);

        assert(report.device_reports[0].device_id == "motion_sensor_1");
        assert(report.device_reports[0].battery_type == "AA");
        assert(report.device_reports[0].battery_life_months == 12.0);
        assert(report.device_reports[0].next_replacement_in_months == 12.0);
        assert(report.device_reports[0].replacements_in_period == 2.0);
        assert(report.device_reports[0].maintenance_cost == 100.0);

        assert(report.device_reports[1].device_id == "door_sensor_1");
        assert(report.device_reports[1].battery_type == "CR2032");
        assert(report.device_reports[1].battery_life_months == 24.0);
        assert(report.device_reports[1].next_replacement_in_months == 24.0);
        assert(report.device_reports[1].replacements_in_period == 1.0);
        assert(report.device_reports[1].maintenance_cost == 120.0);
        assert(report.total_maintenance_cost == 120.0 + 100.0);
    }

    void test_next_replacement_uses_installation_date() {
        using power_planner::model::Device;
        using power_planner::service::BatteryCalculator;

        std::vector<Device> devices{
            {"motion_sensor_1", "motion_sensor", "battery", "AA", 12.0, "2026-01-01", 50.0, 0.0, 0.0, 0.0}
        };

        BatteryCalculator calculator;
        const auto report = calculator.calculate(devices, 12, "2026-06-01");

        assert(report.device_reports.size() == 1);
        assert(report.device_reports[0].next_replacement_in_months == 7.0);
        assert(report.device_reports[0].replacements_in_period == 1);
        assert(report.device_reports[0].maintenance_cost == 50.0);
    }

    void test_mains_devices_are_ignored() {
        using power_planner::model::Device;
        using power_planner::service::BatteryCalculator;

        std::vector<Device> devices{
            {"cam_1", "camera", "mains", "", 0.0, "", 0.0, 5.0, 7.0, 220.0},
            {"motion_sensor_1", "motion_sensor", "battery", "AA", 12.0, "", 0.0, 0.0, 0.0, 0.0}
        };

        BatteryCalculator calculator;

        const auto report = calculator.calculate(devices, 12, "2026-06-01");

        assert(report.device_reports.size() == 1);
        assert(report.period_months == 12);
        assert(report.device_reports[0].device_id == "motion_sensor_1");
        assert(report.device_reports[0].battery_type == "AA");
    }
}

int main() {
    test_battery_devices_are_reported();
    test_next_replacement_uses_installation_date();
    test_mains_devices_are_ignored();
    return 0;
}
