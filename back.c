#include "back.h"
#include <stdio.h>
#include <stdlib.h>

int alloc(int size) {
	int* ptr = (int*)malloc(size);
	if (ptr == NULL) {
		return 1;
	}
	char* s;
	sprintf(s, "%p\n", (void*)ptr);
	return (int)strtol(s, NULL, 16);
}