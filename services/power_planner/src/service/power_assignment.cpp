#include "power_planner/service/power_assignment.hpp"

#include <algorithm>
#include <limits>
#include <unordered_map>

namespace power_planner::service {
    PowerAssignment PowerAssignmentPlanner::plan(
        const std::vector<power_planner::model::Device> &devices,
        const std::vector<power_planner::model::PowerSource> &power_sources) const {
        PowerAssignment assignment;
        std::vector<const power_planner::model::Device *> mains_devices;
        for (const auto &device: devices) {
            if (device.power_type == "mains") {
                mains_devices.push_back(&device);
            }
        }
        std::sort(mains_devices.begin(), mains_devices.end(),
                  [](const auto *left, const auto *right) {
                      return left->peak_power_w > right->peak_power_w;
                  });
        std::unordered_map<std::string, double> free_current;
        for (const auto &source: power_sources) {
            free_current[source.id] = source.max_current_a;
        }
        for (const auto *device: mains_devices) {
            double required_current = device->peak_power_w / device->voltage_v;
            const power_planner::model::PowerSource *best_source = nullptr;
            double best_leftover = std::numeric_limits<double>::max();
            for (const auto &source: power_sources) {
                if (source.voltage_v != device->voltage_v) {
                    continue;
                }
                const double leftover = free_current[source.id] - required_current;
                if (leftover >= 0.0 && leftover < best_leftover) {
                    best_leftover = leftover;
                    best_source = &source;
                }
            }
            if (best_source == nullptr) {
                assignment.unassigned_device_ids.push_back(device->id);
                continue;
            }
            assignment.connections.push_back({device->id, best_source->id});
            free_current[best_source->id] = best_leftover;
        }
        return assignment;
    }
}
