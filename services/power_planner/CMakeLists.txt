cmake_minimum_required(VERSION 3.20)
enable_testing()

project(power_planner VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

include(FetchContent)

FetchContent_Declare(
        nlohmann_json
        URL https://github.com/nlohmann/json/releases/download/v3.11.3/json.tar.xz
)

FetchContent_MakeAvailable(nlohmann_json)

add_executable(power_planner_test
        tests/power_calculator_test.cpp
        src/service/power_calculator.cpp
)

add_executable(battery_calculator_test
        tests/battery_calculator_test.cpp
        src/service/battery_calculator.cpp
)

add_executable(request_loader_test
        tests/request_loader_test.cpp
        src/service/request_loader.cpp
)

add_executable(power_assignment_test
        tests/power_assignment_test.cpp
        src/service/power_assignment.cpp
)

add_executable(result_writer_test
        tests/result_writer_test.cpp
        src/service/result_writer.cpp
)

add_executable(power_planner
        src/main.cpp
        src/service/power_calculator.cpp
        src/service/battery_calculator.cpp
        src/service/request_loader.cpp
        src/service/power_assignment.cpp
        src/service/result_writer.cpp
)

target_include_directories(power_planner_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_include_directories(battery_calculator_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_include_directories(request_loader_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_include_directories(power_assignment_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_include_directories(result_writer_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_include_directories(power_planner PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)


target_link_libraries(power_planner PRIVATE nlohmann_json::nlohmann_json)

target_link_libraries(request_loader_test PRIVATE nlohmann_json::nlohmann_json)

target_link_libraries(result_writer_test PRIVATE nlohmann_json::nlohmann_json)

add_test(NAME power_planner_test COMMAND power_planner_test)
add_test(NAME battery_calculator_test COMMAND battery_calculator_test)
add_test(NAME request_loader_test COMMAND request_loader_test)
add_test(NAME power_assignment_test COMMAND power_assignment_test)
add_test(NAME result_writer_test COMMAND result_writer_test)
set_tests_properties(request_loader_test PROPERTIES
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
)