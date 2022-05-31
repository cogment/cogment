#include "gtest/gtest.h"

#include "slt/settings.h"
#include "slt/settings_context.h"

slt::Setting value = slt::Setting_builder<int>().with_arg("value").with_default(1);

TEST(slt_settings_args, default_values) {
  // Just double-check that the test fixture is behaving correctly

  EXPECT_EQ(value.get(), 1);
  {
    std::vector<const char*> argv = {"doesn't matter", "--value=5"};
    slt::Settings_context ctx("test", argv.size(), argv.data());

    EXPECT_EQ(value.get(), 5);
  }

  EXPECT_EQ(value.get(), 1);
}

TEST(slt_settings_args, invalid_argument) {
  // Just double-check that the test fixture is behaving correctly

  EXPECT_EQ(value.get(), 1);
  {
    std::vector<const char*> argv = {"doesn't matter", "--vale=5"};

    EXPECT_THROW(slt::Settings_context ctx("test", argv.size(), argv.data()), slt::Settings_error);

    EXPECT_EQ(value.get(), 1);
  }

  EXPECT_EQ(value.get(), 1);
}
