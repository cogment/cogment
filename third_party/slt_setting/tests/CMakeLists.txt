include(GoogleTest)
find_package(GTest REQUIRED CONFIG)

add_executable(test_args test_args.cpp)
target_link_libraries(test_args slt_settings GTest::gtest_main GTest::gtest)

gtest_discover_tests(test_args)

