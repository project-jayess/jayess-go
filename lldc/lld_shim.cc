//go:build jayess_lld && cgo

#include "lld_shim.h"

#include <cstdlib>
#include <cstring>
#include <string>
#include <vector>

#include "lld/Common/Driver.h"
#include "llvm/Support/raw_ostream.h"

LLD_HAS_DRIVER(coff)
LLD_HAS_DRIVER(elf)
LLD_HAS_DRIVER(macho)

static char *jayess_lld_copy_message(const std::string &message) {
  char *copy = static_cast<char *>(std::malloc(message.size() + 1));
  if (copy == nullptr) {
    return nullptr;
  }
  std::memcpy(copy, message.c_str(), message.size() + 1);
  return copy;
}

int jayess_lld_link(const char **args, int argc, char **error_message) {
  std::vector<const char *> argv;
  argv.reserve(static_cast<size_t>(argc));
  for (int i = 0; i < argc; i++) {
    argv.push_back(args[i]);
  }

  std::string stdout_text;
  std::string stderr_text;
  llvm::raw_string_ostream stdout_stream(stdout_text);
  llvm::raw_string_ostream stderr_stream(stderr_text);
  lld::DriverDef drivers[] = {
      {lld::WinLink, &lld::coff::link},
      {lld::Gnu, &lld::elf::link},
      {lld::Darwin, &lld::macho::link},
  };

  lld::Result result = lld::lldMain(argv, stdout_stream, stderr_stream, drivers);
  stdout_stream.flush();
  stderr_stream.flush();
  if (result.retCode != 0 && error_message != nullptr) {
    std::string message = stderr_text;
    if (message.empty()) {
      message = stdout_text;
    }
    *error_message = jayess_lld_copy_message(message);
  }
  return result.retCode;
}

void jayess_lld_free_message(char *message) { std::free(message); }
