cmake_minimum_required(VERSION 3.20)
enable_testing()

project(power_planner VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

add_executable(power_planner_test
        tests/power_calculator_test.cpp
        src/service/power_calculator.cpp
)

add_executable(power_planner
        src/main.cpp
        src/service/power_calculator.cpp
        src/service/battery_calculator.cpp
)

target_include_directories(power_planner_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_include_directories(power_planner PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)

add_test(NAME power_planner_test COMMAND power_planner_test)