#ifndef SLT_SETTINGS_SETTINGS_CONTEXT_H
#define SLT_SETTINGS_SETTINGS_CONTEXT_H

#include <string>

namespace slt {
class [[nodiscard]] Settings_context {
 public:
  Settings_context(std::string app_name, int argc, const char* argv[]);
  ~Settings_context();

  bool help_requested() const;
  void print_usage(bool to_stderr) const;

  void validate_all();
 
 private:
  void load_from_args(int argc, const char* argv[]);
  void load_from_env(); 

  std::string app_name_;
};
}  // namespace slt

#endif
