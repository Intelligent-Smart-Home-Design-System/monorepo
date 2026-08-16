#include "power_planner/service/power_calculator.hpp"

#include <unordered_map>
#include <unordered_set>

namespace power_planner::service {
    PowerReport PowerCalculator::calculate(
        const std::vector<power_planner::model::Device> &devices,
        const std::vector<power_planner::model::PowerSource> &power_sources,
        const std::vector<power_planner::model::Connection> &connections)
    const {
        PowerReport report;

        std::unordered_map<std::string, const power_planner::model::Device *> devices_by_id;
        for (const auto &device: devices) {
            if (device.power_type != "mains") {
                continue;
            }
            devices_by_id[device.id] = &device;
            report.total_nominal_power_w += device.nominal_power_w;
            report.total_peak_power_w += device.peak_power_w;
        }

        std::unordered_map<std::string, const power_planner::model::PowerSource *> sources_by_id;
        for (const auto &source: power_sources) {
            sources_by_id[source.id] = &source;
        }

        std::unordered_map<std::string, PowerSourceReport> loads_by_source_id;
        std::unordered_set<std::string> assigned_device_ids;

        for (const auto &connection: connections) {
            const auto device_it = devices_by_id.find(connection.device_id);
            const auto source_it = sources_by_id.find(connection.power_source_id);

            if (device_it == devices_by_id.end() || source_it == sources_by_id.end()) {
                continue;
            }

            const auto &device = *device_it->second;
            const auto &source = *source_it->second;

            auto &source_report = loads_by_source_id[source.id];
            source_report.power_source_id = source.id;
            source_report.nominal_power_w += device.nominal_power_w;
            source_report.peak_power_w += device.peak_power_w;

            assigned_device_ids.insert(device.id);
        }

        for (auto &[source_id, source_report]: loads_by_source_id) {
            const auto &source = *sources_by_id.at(source_id);

            if (source.voltage_v > 0.0) {
                source_report.nominal_current_a = source_report.nominal_power_w / source.voltage_v;
                source_report.peak_current_a = source_report.peak_power_w / source.voltage_v;
            }

            source_report.overloaded_nominal = source_report.nominal_current_a > source.max_current_a;
            source_report.overloaded_peak = source_report.peak_current_a > source.max_current_a;

            report.source_loads.push_back(source_report);
        }

        for (const auto &device: devices) {
            if (device.power_type != "mains") {
                continue;
            }
            if (!assigned_device_ids.contains(device.id)) {
                report.unassigned_device_ids.push_back(device.id);
            }
        }

        return report;
    }
}
