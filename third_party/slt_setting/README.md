# cpp_settings

A straightforward settings library for modern C++.

## Inspiration

This is essentially a modernized version of gflags, with a few added features on top.

## Usage

```
#include "slt/settings.h"

slt::Setting my_setting = slt::Setting_builder<int>()
  .with_default(0)
  .with_description("My setting's info")
  .with_arg("setting")
  .with_env_variable("MY_APP_SETTING");

int main(int argc, const char* argv) {
  {
    slt::Setting_context ctx("my_app", argc, argv);

    int v = my_setting.get();
  }

  // Settings get reset to default when setting contexts are destroyed.

  return 0;
}

```