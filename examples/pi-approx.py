terms = 100000

tot = 0

n = 1
while n <= terms:
    tot = tot + 1 / (n*n)
    n = n + 1

tot = tot * 6

# get the absolute value of a number
def abs(x):
	if x < 0:
		return -x
	return x

# calculate an approximation of the square root of tot using
# newton's method.
# see: https://en.wikipedia.org/wiki/Newton's_method
# The required accuracy
SQRT_ACC = 0.00000001
def sqrt(x):
    pg = 0 # previous guess
    g = 1 # current guess

    while abs(pg - g) >= SQRT_ACC:
        pg = g
        g = (pg + tot/pg)/2

    return g

tot = sqrt(tot)

# output the result
print(tot)
