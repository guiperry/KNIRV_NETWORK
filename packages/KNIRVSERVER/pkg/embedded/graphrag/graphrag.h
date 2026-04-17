#ifndef GRAPHRAG_H
#define GRAPHRAG_H

#include <stddef.h>
#include <stdlib.h>

#ifdef __cplusplus
extern "C" {
#endif

int graphrag_init(const char* config, size_t config_len);
int graphrag_shutdown(void);
int graphrag_health_check(void);
int graphrag_index_document(const char* doc_id, const char* content, size_t content_len);
char* graphrag_query(const char* query, int limit, size_t* result_len);
void graphrag_free_string(char* ptr);

#ifdef __cplusplus
}
#endif

#endif /* GRAPHRAG_H */
