#include "back.h"
#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>

int vasprintf(char **strp, const char *fmt, va_list ap) {
	int len = _vscprintf(fmt, ap);
	if (len == -1)
		return -1;
	size_t size = (size_t)len + 1;
	char *str = malloc(size);
	if (!str)
		return -1;
	int r = vsprintf_s(str, len + 1, fmt, ap);
	if (r == -1) {
		free(str);
		return -1;
	}
	*strp = str;
	return r;
}

int asprintf(char **strp, const char *fmt, ...) {
	va_list ap;
	va_start(ap, fmt);
	int r = vasprintf(strp, fmt, ap);
	va_end(ap);
	return r;
}

unsigned long long alloc(int size) {
	int* ptr = (int*)malloc(size);
	if (ptr == NULL)
		return 1;
	char *s;
	if (asprintf(&s, "%llu", (unsigned long long)ptr) == -1)
		return 1;
	unsigned long long val = 0;
	sscanf(s, "%llu", &val);
	return val;
}
