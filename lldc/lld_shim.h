#ifndef JAYESS_LLD_SHIM_H
#define JAYESS_LLD_SHIM_H

#ifdef __cplusplus
extern "C" {
#endif

int jayess_lld_link(const char **args, int argc, char **error_message);
void jayess_lld_free_message(char *message);

#ifdef __cplusplus
}
#endif

#endif
