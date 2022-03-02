#include "slt/settings.h"
#include "slt/settings_context.h"

#include <string>

slt::Setting ip_address = slt::Setting_builder<std::string>()
                              .with_default("localhost")
                              .with_description("The IP address to operate on")
                              .with_arg("ip")
                              .with_env_variable("IP");

int main(int argc, const char* argv[]) {
  slt::Settings_context ctx("hello_settings", argc, argv);

  if (ctx.help_requested()) {
    return 0;
  }

  std::cout << "the value of ip_address is: " << ip_address.get() << "\n";

  return 0;
}