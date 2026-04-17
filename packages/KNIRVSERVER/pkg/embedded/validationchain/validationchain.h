#ifndef VALIDATIONCHAIN_H
#define VALIDATIONCHAIN_H

#include <stddef.h>
#include <stdlib.h>

#ifdef __cplusplus
extern "C" {
#endif

int validation_chain_init(const char* config, size_t config_len);
int validation_chain_shutdown(void);
int validation_chain_health_check(void);
char* validation_chain_validate(const char* tx_data, size_t tx_len, int* result_code, size_t* result_len);

#ifdef __cplusplus
}
#endif

#endif /* VALIDATIONCHAIN_H */
