#include "reduce.h"
#include "atomicf.h"
#include "float3.h"

#define load_vecnorm2(i) \
	sqr(x[i]) + sqr(y[i]) +  sqr(z[i])

extern "C" __global__ void
reducemaxvecnorm2(float* __restrict__ x, float* __restrict__ y, float* __restrict__ z, float* __restrict__ dst, float initVal, int n) {
	reduce(load_vecnorm2, fmax, atomicFmaxabs)
}

