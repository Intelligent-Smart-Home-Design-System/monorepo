#include "power_planner/service/request_loader.hpp"

#include <fstream>

#include <nlohmann/json.hpp>

#include "power_planner/model/device.hpp"
#include "power_planner/model/power_source.hpp"
#include "power_planner/model/connection.hpp"

namespace power_planner::service {
    power_planner::model::PlannerRequest RequestLoader::load(const std::string &file_path) const {
        std::ifstream input(file_path);
        nlohmann::json json;
        input >> json;
        power_planner::model::PlannerRequest request;
        for (const auto &device_from_json: json["input"]["devices"]) {
            power_planner::model::Device device;
            device.id = device_from_json.value("id", "");
            device.type = device_from_json.value("type", "");
            device.power_type = device_from_json.value("power_type", "");
            device.battery_type = device_from_json.value("battery_type", "");
            device.battery_life_months = device_from_json.value("battery_life_months", 0.0);
            device.battery_installed_at = device_from_json.value("battery_installed_at", "");
            device.battery_price = device_from_json.value("battery_price", 0.0);
            device.nominal_power_w = device_from_json.value("nominal_power_w", 0.0);
            device.peak_power_w = device_from_json.value("peak_power_w", 0.0);
            device.voltage_v = device_from_json.value("voltage_v", 220.0);
            request.input.devices.push_back(device);
        }
        for (const auto &source_from_json: json["input"]["power_sources"]) {
            power_planner::model::PowerSource source;
            source.id = source_from_json.value("id", "");
            source.kind = source_from_json.value("kind", "");
            source.max_current_a = source_from_json.value("max_current_a", 0.0);
            source.voltage_v = source_from_json.value("voltage_v", 220.0);
            request.input.power_sources.push_back(source);
        }
        for (const auto &connection_from_json: json["input"]["connections"]) {
            power_planner::model::Connection connection;
            connection.device_id = connection_from_json.value("device_id", "");
            connection.power_source_id = connection_from_json.value("power_source_id", "");
            request.input.connections.push_back(connection);
        }
        request.options.battery_period_months = json["options"].value("battery_period_months", 12);
        request.options.current_date = json["options"].value("current_date", "");
        return request;
    }
}
